package _type

import "net"

type PendingConnection struct {
	Conn             net.Conn
	ConnectedMessage Message
}

type Message struct {
	ClientID        string
	Message         string
	NumberOfPlayers int // 0 - со всеми , 1 - соло , 2 - дуо , 3 - трио
	MapName         string
	AppVersion      string
}

type RoomSettings struct {
	ID         int
	CurrentMap string
	AppVersion string
	MaxPlayers int
}

type Response struct {
	ID         int    `json:"-"`
	Status     string `json:"status"`
	IP         string `json:"ip"`
	Port       int    `json:"-"`
	MapName    string `json:"map_name"`
	AppVersion string `json:"-"`
}
