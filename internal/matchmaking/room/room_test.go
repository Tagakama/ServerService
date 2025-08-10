package room_test

import (
	"github.com/Tagakama/ServerManager/internal/matchmaking/matchmaker"
	ro "github.com/Tagakama/ServerManager/internal/matchmaking/room"
	_type "github.com/Tagakama/ServerManager/internal/tcp-server/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRoom_ClosesAfterTimeout(t *testing.T) {
	closed := make(chan bool, 1)

	// Установим таймер на 100 мс для быстрого теста
	room, err := ro.New(_type.RoomSettings{
		ID:         1,
		MaxPlayers: 8,
		CurrentMap: "TestMap",
		AppVersion: "v1.0",
	})
	require.NoError(t, err)

	room.Timeout = 100 * time.Millisecond
	room.Timer = time.NewTimer(room.Timeout)

	// Отслеживаем закрытие
	room.OnComplete = func(r *ro.Room) {
		closed <- true
	}

	// Запускаем таймер вручную (имитируем поведение New)
	go func(r *ro.Room) {
		<-r.Timer.C
		r.Mutex.Lock()
		defer r.Mutex.Unlock()
		if !r.Closed {
			r.Closed = true
			if r.OnComplete != nil {
				r.OnComplete(r)
			}
		}
	}(room)

	// Проверим, что закроется в течение 500 мс
	select {
	case <-closed:
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Room did not close after timeout")
	}

	assert.True(t, room.Closed, "Room should be closed by timer")
}

func TestRoom_RemovedAfterTimeout(t *testing.T) {
	mm := matchmaker.New()
	done := make(chan struct{}, 1)

	conn := &_type.PendingConnection{
		ConnectedMessage: _type.Message{
			ClientID:        "test-client",
			MapName:         "timeout-map",
			AppVersion:      "v1",
			NumberOfPlayers: 1,
		},
	}

	mm.AddNewRoom(conn)
	require.Len(t, mm.CurrentRooms, 1)
	room := mm.CurrentRooms[0]

	room.Timeout = 100 * time.Millisecond
	room.Timer = time.NewTimer(room.Timeout)
	room.OnComplete = func(r *ro.Room) {
		mm.RemoveRoom(r)
		done <- struct{}{}
	}

	go func(r *ro.Room) {
		<-r.Timer.C
		r.Mutex.Lock()
		defer r.Mutex.Unlock()
		if !r.Closed {
			r.Closed = true
			if r.OnComplete != nil {
				r.OnComplete(r)
			}
		}
	}(room)

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Room was not closed and removed after timeout")
	}

	assert.Len(t, mm.CurrentRooms, 0, "Room should be removed from matchmaker")
}
