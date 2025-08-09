package matchmaker

import (
	"encoding/json"
	"errors"
	"fmt"
	server_launcher "github.com/Tagakama/ServerManager/internal/game-server/server-launcher"
	r "github.com/Tagakama/ServerManager/internal/matchmaking/room"
	_type "github.com/Tagakama/ServerManager/internal/tcp-server/type"
	"sync"
	"time"
)

type RoomCloser interface {
	RemoveRoom(closedRoom *r.Room)
	RemoveClosedRoom()
}

type Matchmaker struct {
	CurrentRooms []*r.Room
	mu           sync.Mutex
	launcher     *server_launcher.ServerLauncher
}

var roomsCount = 1

func NewMatchmaker(launcher *server_launcher.ServerLauncher) *Matchmaker {
	return &Matchmaker{
		CurrentRooms: make([]*r.Room, 0),
		mu:           sync.Mutex{},
		launcher:     launcher,
	}
}

func (m *Matchmaker) AddNewRoom(connection *_type.PendingConnection) error {
	newRoomSettings := _type.RoomSettings{
		ID:         roomsCount,
		MaxPlayers: 8,
		CurrentMap: connection.ConnectedMessage.MapName,
		AppVersion: connection.ConnectedMessage.AppVersion,
	}
	newRoom, err := r.New(newRoomSettings)
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating new room :%s", err))
	}

	newRoom.OnComplete = func(r *r.Room) {
		m.RoomCopmlete(r)
	}

	//m.mu.Lock()
	m.CurrentRooms = append(m.CurrentRooms, newRoom)
	roomsCount++
	//m.mu.Unlock()
	return nil
}

func (m *Matchmaker) InviteInRoom(connection *_type.PendingConnection) {
	playersCount := connection.ConnectedMessage.NumberOfPlayers

	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeClosedRoomLocked()

	added := false

	for _, room := range m.CurrentRooms {
		if room.CheckingFreeSpace(playersCount) {
			room.AddPlayer(connection)
			added = true
			break
		}
	}

	// ❗ Только если не добавили — создаём новую комнату
	if !added {
		m.addAndAssign(connection)
	}
}

func (m *Matchmaker) removeClosedRoomLocked() {
	var activeRooms []*r.Room
	for _, room := range m.CurrentRooms {
		if !room.Closed {
			activeRooms = append(activeRooms, room)
		}
	}
	m.CurrentRooms = activeRooms
}

func (m *Matchmaker) RemoveClosedRoom() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeClosedRoomLocked()
}

func (m *Matchmaker) RemoveRoom(closedRoom *r.Room) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var updatedRooms []*r.Room
	for _, room := range m.CurrentRooms {
		if room != closedRoom {
			updatedRooms = append(updatedRooms, room)
		}
	}
	m.CurrentRooms = updatedRooms
}

func (m *Matchmaker) addAndAssign(connection *_type.PendingConnection) {
	err := m.AddNewRoom(connection)
	if err != nil {
		fmt.Errorf("Error adding new room :%s", err)
	}
	lastRoom := m.CurrentRooms[len(m.CurrentRooms)-1]
	lastRoom.AddPlayer(connection)
}

func (m *Matchmaker) RoomCopmlete(r *r.Room) {

	m.launcher.LaunchGameServer(r)

	time.Sleep(500 * time.Millisecond)

	m.SendResponse(r)

	m.removeClosedRoomLocked()
}

func (m *Matchmaker) SendResponse(r *r.Room) {
	unicName := fmt.Sprintf("%s%s%d", r.AppVersion, r.CurrentMap, r.ID)

	newResponse := _type.Response{
		Status:  "running",
		IP:      unicName,
		MapName: r.CurrentMap,
	}

	response, err := json.Marshal(newResponse)
	if err != nil {
		fmt.Errorf("Error marshalling response :%s", err)
	}

	for _, player := range r.Players {
		fmt.Fprintf(player.Conn, string(response))
	}
}
