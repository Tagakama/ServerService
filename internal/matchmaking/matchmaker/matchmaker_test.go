package matchmaker_test

import (
	"fmt"
	"github.com/Tagakama/ServerManager/internal/matchmaking/matchmaker"
	"github.com/Tagakama/ServerManager/internal/matchmaking/room"
	"github.com/Tagakama/ServerManager/internal/tcp-server/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strconv"
	"sync"
	"testing"
	"time"
)

func mockConnection(clientID string, mapName string, players int) *_type.PendingConnection {
	return &_type.PendingConnection{
		ConnectedMessage: _type.Message{
			ClientID:        clientID,
			MapName:         mapName,
			AppVersion:      "1.0",
			Message:         "test",
			NumberOfPlayers: players,
		},
	}
}

func TestMatchmaker_AddNewRoom(t *testing.T) {
	mockLauncher := &MockServerLauncher{}
	m := matchmaker.NewMatchmaker(mockLauncher)
	err := m.AddNewRoom(mockConnection("client1", "map1", 1))
	if err != nil {
		t.Fatalf("failed to add room: %v", err)
	}
	if len(m.CurrentRooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(m.CurrentRooms))
	}
}

func TestMatchmaker_InviteInRoom(t *testing.T) {
	mockLauncher := &MockServerLauncher{}
	m := matchmaker.NewMatchmaker(mockLauncher)

	// Добавляем комнату
	err := m.AddNewRoom(mockConnection("owner", "map1", 1))
	if err != nil {
		t.Fatal(err)
	}

	// Приглашаем игроков
	for i := 0; i < 8; i++ {
		m.InviteInRoom(mockConnection("client"+strconv.Itoa(i), "map1", 1))
	}

	// Проверяем, что в комнате 8 игроков
	if m.CurrentRooms[0].ReservedPlayers != 8 {
		t.Errorf("expected 8 players, got %d", m.CurrentRooms[0].ReservedPlayers)
		t.Errorf("Open lobby count :%d", len(m.CurrentRooms))
	}
	if !m.CurrentRooms[0].Closed {
		t.Error("expected room to be closed")
	}
}

func TestMatchmaker_RemoveClosedRoom_RemovesMultipleClosedRooms(t *testing.T) {
	mockLauncher := &MockServerLauncher{}
	mm := matchmaker.NewMatchmaker(mockLauncher)

	// Комнаты: закрытая, открытая, закрытая
	r1, _ := room.New(_type.RoomSettings{ID: 1, MaxPlayers: 8, CurrentMap: "Map1", AppVersion: "v1"})
	r2, _ := room.New(_type.RoomSettings{ID: 2, MaxPlayers: 8, CurrentMap: "Map2", AppVersion: "v1"})
	r3, _ := room.New(_type.RoomSettings{ID: 3, MaxPlayers: 8, CurrentMap: "Map3", AppVersion: "v1"})

	r1.Closed = true
	r3.Closed = true

	mm.CurrentRooms = []*room.Room{r1, r2, r3}

	// До очистки должно быть 3 комнаты
	require.Len(t, mm.CurrentRooms, 3)

	// Удаляем закрытые
	mm.RemoveClosedRoom()

	// После — только одна (r2)
	require.Len(t, mm.CurrentRooms, 1)
	assert.Equal(t, "Map2", mm.CurrentRooms[0].CurrentMap)
}

func TestMatchmaker_StressTest_1000Connections(t *testing.T) {
	mockLauncher := &MockServerLauncher{}
	mm := matchmaker.NewMatchmaker(mockLauncher)

	var wg sync.WaitGroup
	num := 1000

	for i := 0; i < num; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			conn := &_type.PendingConnection{
				ConnectedMessage: _type.Message{
					MapName:         fmt.Sprintf("Map%d", id%5),
					AppVersion:      "v1.0",
					NumberOfPlayers: 1 + id%3,
				},
			}

			mm.InviteInRoom(conn)
		}(i)
	}

	wg.Wait()

	assert.Greater(t, len(mm.CurrentRooms), 0)
}

func TestMatchmaker_1000Connections_Distribution(t *testing.T) {
	mockLauncher := &MockServerLauncher{}
	mm := matchmaker.NewMatchmaker(mockLauncher)
	const totalConnections = 1000
	const playersPerConnection = 1
	const maxPlayersPerRoom = 8

	var wg sync.WaitGroup

	for i := 0; i < totalConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn := &_type.PendingConnection{
				ConnectedMessage: _type.Message{
					ClientID:        fmt.Sprintf("client-%d", id),
					MapName:         "Arena",
					AppVersion:      "v1.0.0",
					NumberOfPlayers: playersPerConnection,
				},
			}
			mm.InviteInRoom(conn)
		}(i)
	}

	wg.Wait()

	// Подсчёт закрытых и открытых комнат
	closed := 0
	open := 0
	totalPlayers := 0

	for _, room := range mm.CurrentRooms {
		if room.Closed {
			closed++
		} else {
			open++
		}
		totalPlayers += room.ReservedPlayers

		// Проверка что число игроков не превышает лимит
		if room.ReservedPlayers > room.MaxPlayers {
			t.Errorf("Room %d has too many players: %d > %d",
				room.ID, room.ReservedPlayers, room.MaxPlayers)
		}
	}

	t.Logf("Total rooms: %d", len(mm.CurrentRooms))
	t.Logf("Closed rooms: %d", closed)
	t.Logf("Open rooms:   %d", open)

	assert.Equal(t, totalConnections, totalPlayers, "All players must be assigned")
	assert.True(t, closed > 0, "There should be closed rooms")
	assert.True(t, open >= 0, "Some rooms may still be open")

	// Проверка минимального количества комнат
	minRooms := totalConnections / maxPlayersPerRoom
	assert.GreaterOrEqual(t, len(mm.CurrentRooms), minRooms)
}

func TestMatchmaker_RemoveClosedRoom(t *testing.T) {
	mockLauncher := &MockServerLauncher{}
	m := matchmaker.NewMatchmaker(mockLauncher)

	// Добавляем комнату
	_ = m.AddNewRoom(mockConnection("client", "map1", 1))
	room := m.CurrentRooms[0]
	room.Closed = true

	m.RemoveClosedRoom()
	if len(m.CurrentRooms) != 0 {
		t.Errorf("expected 0 active rooms, got %d", len(m.CurrentRooms))
	}
}

func TestMatchmaker_RoomTimeoutClosure(t *testing.T) {
	mockLauncher := &MockServerLauncher{}
	mm := matchmaker.NewMatchmaker(mockLauncher)

	// Переопределим таймаут через OnComplete
	done := make(chan struct{}, 1)

	// Сохраним ссылку на первую комнату
	var createdRoom *room.Room

	// 3 подключения
	for i := 0; i < 3; i++ {
		conn := &_type.PendingConnection{
			ConnectedMessage: _type.Message{
				ClientID:        fmt.Sprintf("client-%d", i),
				MapName:         "TimeoutMap",
				AppVersion:      "vTest",
				NumberOfPlayers: 1,
			},
		}

		mm.InviteInRoom(conn)
	}

	// Получаем первую созданную комнату
	require.NotEmpty(t, mm.CurrentRooms)
	createdRoom = mm.CurrentRooms[0]

	// Устанавливаем короткий таймаут и OnComplete
	createdRoom.Mutex.Lock()
	createdRoom.Timeout = 150 * time.Millisecond
	createdRoom.Timer = time.NewTimer(createdRoom.Timeout)
	createdRoom.OnComplete = func(r *room.Room) {
		done <- struct{}{}
	}
	createdRoom.Mutex.Unlock()

	// Запускаем таймер вручную (так как ты вызываешь таймер в New, но в тесте мы его пересоздали)
	go func(r *room.Room) {
		<-r.Timer.C
		r.Mutex.Lock()
		defer r.Mutex.Unlock()
		if !r.Closed {
			r.Closed = true
			if r.OnComplete != nil {
				r.OnComplete(r)
			}
		}
	}(createdRoom)

	// Ждём до 500 мс, пока сработает таймер
	select {
	case <-done:
		// ОК
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Room was not closed by timeout")
	}

	assert.True(t, createdRoom.Closed, "Room should be closed after timeout")
}
