package room

import (
	"errors"
	"fmt"
	_type "github.com/Tagakama/ServerManager/internal/tcp-server/type"
	"sync"
	"time"
)

type Room struct {
	ID              int
	Players         []*_type.PendingConnection
	CurrentMap      string
	AppVersion      string
	SessionName     string
	ReservedPlayers int
	MaxPlayers      int
	Timer           *time.Timer
	Timeout         time.Duration
	Closed          bool
	Mutex           sync.Mutex
	OnComplete      func(room *Room)
}

func New(settings _type.RoomSettings) (*Room, error) {
	if settings.ID <= 0 {
		return &Room{}, errors.New("Room ID is incorrect")
	}

	room := &Room{
		ID:          settings.ID,
		Players:     make([]*_type.PendingConnection, 0),
		CurrentMap:  settings.CurrentMap,
		AppVersion:  settings.AppVersion,
		MaxPlayers:  settings.MaxPlayers,
		SessionName: fmt.Sprintf("%s_%d_%s", settings.AppVersion, settings.ID, settings.CurrentMap),
		Mutex:       sync.Mutex{},
		Timer:       time.NewTimer(30 * time.Second),
		Timeout:     time.Duration(30 * time.Second),
		Closed:      false,
	}

	go func(r *Room) {
		<-r.Timer.C
		r.Mutex.Lock()
		defer r.Mutex.Unlock()

		if r.Closed {
			return
		}
		r.Closed = true
		if r.OnComplete != nil {
			go r.OnComplete(r)
		}
	}(room)
	fmt.Printf("New Room ID: %d\n", room.ID)
	return room, nil
}

func (room *Room) CheckingFreeSpace(playerCount int) bool {
	return room.MaxPlayers-room.ReservedPlayers >= playerCount
}

func (room *Room) AddPlayer(player *_type.PendingConnection) {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	room.Players = append(room.Players, player)
	room.ReservedPlayers += player.ConnectedMessage.NumberOfPlayers
	fmt.Printf("Player %s, connected to room %d\n", player.ConnectedMessage.ClientID, room.ID)
	if room.ReservedPlayers == room.MaxPlayers {
		room.Closed = true
	}
}
