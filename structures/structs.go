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
	PlayerPoints       map[Player]int
	PlayersPlayed      map[Player]bool
	PlayerLetters      map[Player]string
}

type Player struct {
	Nickname string
	Socket   net.Conn
}

type DictionaryItem struct {
	Sentence string
	Hint     string
}
