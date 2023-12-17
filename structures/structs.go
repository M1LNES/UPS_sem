package structures

import "net"

type Game struct {
	ID        string
	Players   map[int]Player
	GameData  GameState
	TurnIndex int
}

type GameState struct {
	IsLobby            bool
	SentenceToGuess    string
	CharactersSelected []string
}

type Player struct {
	Nickname string
	Socket   net.Conn
}
