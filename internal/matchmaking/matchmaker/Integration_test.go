package matchmaker_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Tagakama/ServerManager/internal/matchmaking/matchmaker"
	"github.com/Tagakama/ServerManager/internal/matchmaking/room"
	"github.com/Tagakama/ServerManager/internal/tcp-server/type"
	"github.com/Tagakama/ServerManager/internal/tcp-server/workers"
)

type MockServerLauncher struct {
	mu       sync.Mutex
	launched []*room.Room
}

func (m *MockServerLauncher) LaunchGameServer(r *room.Room) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.launched = append(m.launched, r)
}

func (m *MockServerLauncher) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.launched)
}

func TestMatchmaker_Distribution(t *testing.T) {
	const totalConnections = 1000
	const maxPlayers = 8
	const workerCount = 8

	// Мок лаунчера
	mockLauncher := &MockServerLauncher{}

	// Создаём матчмейкер с моком
	mm := &matchmaker.Matchmaker{
		CurrentRooms: make([]*room.Room, 0),
		Launcher:     mockLauncher,
	}

	// Создаём пул воркеров
	wp := workers.NewWorkerPool(workerCount, mm)

	// Запускаем 1000 подключений
	var wg sync.WaitGroup
	wg.Add(totalConnections)
	for i := 0; i < totalConnections; i++ {
		go func(id int) {
			defer wg.Done()
			conn := &_type.PendingConnection{
				ConnectedMessage: _type.Message{
					ClientID:        fmt.Sprintf("client-%d", id),
					MapName:         "Arena",
					AppVersion:      "v1",
					NumberOfPlayers: 1,
				},
			}
			wp.AddTask(conn)
		}(i)
	}

	// Ждём выполнения
	wg.Wait()
	time.Sleep(500 * time.Millisecond) // даём воркерам завершить

	// Проверяем
	totalPlayers := 0
	closedRooms := 0
	for _, r := range mm.CurrentRooms {
		totalPlayers += r.ReservedPlayers
		if r.Closed {
			closedRooms++
		}
	}

	assert.Equal(t, totalConnections, totalPlayers, "Все игроки должны быть распределены")
	require.GreaterOrEqual(t, closedRooms, totalConnections/maxPlayers, "Должно быть достаточно закрытых комнат")
}

func TestRoom_Timeout(t *testing.T) {
	mockLauncher := &MockServerLauncher{}
	mm := &matchmaker.Matchmaker{
		CurrentRooms: make([]*room.Room, 0),
		Launcher:     mockLauncher,
	}

	// Создаём комнату с таймаутом 1 сек
	settings := _type.RoomSettings{
		ID:         1,
		MaxPlayers: 8,
		CurrentMap: "Arena",
		AppVersion: "v1",
	}
	r, err := room.New(settings)
	require.NoError(t, err)
	r.Timeout = time.Second
	r.Timer.Reset(r.Timeout)

	closedCh := make(chan struct{})
	r.OnComplete = func(_ *room.Room) {
		close(closedCh)
	}

	mm.CurrentRooms = append(mm.CurrentRooms, r)

	// Добавляем 3 игрока, комната не закрыта
	for i := 0; i < 3; i++ {
		conn := &_type.PendingConnection{
			ConnectedMessage: _type.Message{
				ClientID:        fmt.Sprintf("client-%d", i),
				MapName:         "Arena",
				AppVersion:      "v1",
				NumberOfPlayers: 1,
			},
		}
		r.AddPlayer(conn)
	}

	require.False(t, r.Closed, "Комната не должна быть закрыта сразу")

	// Ждём таймаут
	select {
	case <-closedCh:
		assert.True(t, r.Closed, "Комната должна закрыться по таймауту")
	case <-time.After(2 * time.Second):
		t.Fatal("Таймаут не сработал за ожидаемое время")
	}
}
