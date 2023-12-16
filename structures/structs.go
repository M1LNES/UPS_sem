package structures

import "net"

type Game struct {
	ID       string
	Players  map[int]Player
	GameData GameState
}

type GameState struct {
	//ID      string
	//Players map[int]Player
	IsLobby bool
}

type Player struct {
	Nickname string
	Socket   net.Conn
}
