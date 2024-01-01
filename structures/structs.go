package structures

import "net"

type Game struct {
	ID       string
	Players  map[int]Player
	GameData GameState
}

type GameState struct {
	IsLobby            bool
	SentenceToGuess    string
	Hint               string
	CharactersSelected []string
	PlayerPoints       map[string]int
	PlayersPlayed      map[string]bool
	PlayerLetters      map[string]string
}

type Player struct {
	Nickname    string
	Socket      net.Conn
	PingCounter int
}

type DictionaryItem struct {
	Sentence string
	Hint     string
}
