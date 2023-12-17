package main

import (
	"UPS_sem/constants"
	"UPS_sem/structures"
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	connHost = constants.ConnHost
	connPort = constants.ConnPort
	connType = constants.ConnType
)

var clientsMap = make(map[net.Conn]structures.Player)
var gameMap = make(map[string]structures.Game)
var (
	dictionary      []string
	dictionaryMutex sync.Mutex
)
var clientsMapMutex sync.Mutex
var gameMapMutex sync.Mutex

func main() {
	initialiseGameMap()
	createDictionary()

	socket, err := net.Listen(connType, connHost+":"+connPort)

	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer socket.Close()

	for {
		client, err := socket.Accept()
		if err != nil {
			fmt.Println("Error on Accept", err.Error())
			return
		}

		fmt.Println("Client " + client.RemoteAddr().String() + " connected.")

		go handleConnection(client)
	}
}

func createDictionary() {
	content, _ := ioutil.ReadFile("dictionary/" + constants.DictionaryFile)

	sentenceArray := strings.Split(string(content), "\n")

	var cleanedSentences []string
	for _, sentence := range sentenceArray {
		if sentence != "" {
			cleanedSentences = append(cleanedSentences, sentence)
		}
	}

	dictionaryMutex.Lock()
	dictionary = cleanedSentences
	dictionaryMutex.Unlock()
}

func initialiseGameMap() {
	gameMapMutex.Lock()
	for i := 1; i <= constants.RoomsCount; i++ {
		lobbyID := fmt.Sprintf("lobby%d", i)
		gameMap[lobbyID] = structures.Game{
			ID:      lobbyID,
			Players: make(map[int]structures.Player),
			GameData: structures.GameState{
				IsLobby: true,
			},
			TurnIndex: 0,
		}
	}
	gameMapMutex.Unlock()
}

func printGameMap() {
	fmt.Printf("Printing gaming lobbies: \n")
	gameMapMutex.Lock()
	for lobbyID, game := range gameMap {
		fmt.Printf("Lobby ID: %s\n isLobby:%b", lobbyID, game.GameData.IsLobby)
		fmt.Printf("Number of Players: %d\n", len(game.Players))
	}
	gameMapMutex.Unlock()
	fmt.Printf("Printing main lobby: \n")
	for client := range clientsMap {
		fmt.Printf("Client: %s, Username: %s\n", client.RemoteAddr(), clientsMap[client].Nickname)
	}
}

func handleConnection(client net.Conn) {
	defer client.Close()

	reader := bufio.NewReader(client)

	for {
		readBuffer, err := reader.ReadBytes('\n')
		if err != nil {
			clientsMapMutex.Lock()
			fmt.Println("Zabijim: ", clientsMap[client].Nickname)
			clientsMapMutex.Unlock()
			fmt.Println("Client disconnected.:", client)
			return
		}

		// Convert to string and remove trailing newline characters
		message := strings.TrimRight(string(readBuffer), "\r\n")
		fmt.Println("Msg:", message)

		if isLengthValid(message) {
			fmt.Println("Message structure is valid.")
			handleMessage(message, client)
		} else {
			fmt.Println("Message structure is invalid. Closing connection.")
			return
		}

		// Check if the client wants to close the connection
		if message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:] == "bye" {
			fmt.Println("Closing connection based on client request.")
			return
		}

		// Echo the message back to the client
		//client.Write(readBuffer)
	}
}
func findPlayerBySocket(client net.Conn) bool {
	gameMapMutex.Lock()
	defer gameMapMutex.Unlock()
	for _, gameState := range gameMap {
		for _, player := range gameState.Players {
			if player.Socket == client {
				return true
			}
		}
	}
	return false
}

func findPlayerBySocketReturn(client net.Conn) *structures.Player {
	gameMapMutex.Lock()
	defer gameMapMutex.Unlock()
	for _, gameState := range gameMap {
		for _, player := range gameState.Players {
			if player.Socket == client {
				return &player
			}
		}
	}
	return nil
}

func handleMessage(message string, client net.Conn) {
	clientsMapMutex.Lock()
	if _, exists := clientsMap[client]; !exists && findPlayerBySocket(client) == false {
		if createNickForConnection(client, message) {
			fmt.Println("Client successfully added, his name: ", clientsMap[client].Nickname)
		} else {
			fmt.Println("Firstly you must identify yourself, aborting!")
			client.Close()
		}
	} else {

		messageType := message[len(constants.MessageHeader)+constants.MessageLengthFormat : len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength]
		extractedMessage := message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:]

		switch messageType {
		case "join":
			joinPlayerIntoGame(client, message)
		case "info":
			printGameMap()
		case "play":
			startTheGame(client, extractedMessage)
		case "lett":
			receiveLetter(client, extractedMessage)
		default:
			fmt.Println("Unknown command ", messageType)
		}
	}

	clientsMapMutex.Unlock()
}

func receiveLetter(client net.Conn, message string) {

	if findPlayerBySocketReturn(client) == nil {
		fmt.Println("Nenasel jsem daneho hrace, koncim")
		return
	}

	//fmt.Println("Lobby ID a zbytek: ", findLobbyWithPlayer(*findPlayerBySocketReturn(client)).ID, findLobbyWithPlayer(*findPlayerBySocketReturn(client)))
	lobbyID := findLobbyWithPlayer(*findPlayerBySocketReturn(client)).ID
	lobby, ok := gameMap[lobbyID]
	if ok {
		fmt.Println("Hadas vetu: ", lobby.GameData.SentenceToGuess)
		gameMapMutex.Lock()
		if contains(lobby.GameData.CharactersSelected, message) {
			fmt.Println("Uz obsahuje")
		} else {
			fmt.Println("Pridavam novy prvek.")
			lobby.GameData.CharactersSelected = append(lobby.GameData.CharactersSelected, message)
			gameMap[lobbyID] = lobby
		}

		if isWordGuessed(&lobby) {
			fmt.Println("Uhodl jsi celou vetu")
			movePlayersBackToMainLobby(&lobby)
			lobby.GameData.IsLobby = true
			gameMap[lobbyID] = lobby
		}
		gameMapMutex.Unlock()
	}

}

func movePlayersBackToMainLobby(game *structures.Game) {
	for _, player := range game.Players {
		_, err := player.Socket.Write([]byte("Game over, moving you into lobby\n"))
		if err != nil {
			fmt.Println("Error writing to server:", err.Error())
		}
		clientsMap[player.Socket] = player
	}

	game.Players = make(map[int]structures.Player)
}

func isWordGuessed(lobby *structures.Game) bool {
	sentenceToGuess := strings.ToLower(lobby.GameData.SentenceToGuess)
	charactersSelected := strings.ToLower(strings.Join(lobby.GameData.CharactersSelected, ""))
	fmt.Printf("Characters selected %s Sentence %s \n", charactersSelected, sentenceToGuess)
	for _, char := range sentenceToGuess {
		if !unicode.IsLetter(char) {
			continue // Skip non-letter characters
		}

		if !strings.ContainsRune(charactersSelected, unicode.ToLower(char)) {
			return false
		}
	}
	return true
}

func contains(slice []string, element string) bool {
	for _, el := range slice {
		if el == element {
			return true
		}
	}
	return false
}

func startTheGame(client net.Conn, message string) {
	player := findPlayerBySocketReturn(client)
	if player == nil {
		fmt.Println("Nenasel jsem daneho hrace, koncim")
		return
	}
	lobby := findLobbyWithPlayer(*player)
	if canLobbyBeStarted(*lobby) {
		switchLobbyToGame(lobby.ID)
	} else {
		fmt.Println("Could not switch to game - not enough players yet.")
	}
}

func canLobbyBeStarted(lobby structures.Game) bool {
	gameMapMutex.Lock()
	defer gameMapMutex.Unlock()
	return len(lobby.Players) > 1 && lobby.GameData.IsLobby
}

func findLobbyWithPlayer(player structures.Player) *structures.Game {
	gameMapMutex.Lock()
	defer gameMapMutex.Unlock()
	for _, game := range gameMap {
		for _, p := range game.Players {
			if p == player {
				return &game
			}
		}
	}
	return nil
}

func switchLobbyToGame(lobbyID string) {
	fmt.Println("Updatuju lobby: ", lobbyID)
	gameMapMutex.Lock()
	defer gameMapMutex.Unlock()

	if existingGame, ok := gameMap[lobbyID]; ok {
		existingGame.GameData.IsLobby = false
		existingGame.GameData.SentenceToGuess = selectRandomSentence()
		existingGame.GameData.CharactersSelected = []string{}
		gameMap[lobbyID] = existingGame
		for _, player := range gameMap[lobbyID].Players {
			player.Socket.Write([]byte("hej vole, hrajes\n"))
		}
		return
	}
}

func selectRandomSentence() string {
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(dictionary))
	return dictionary[index]
}

func isLobbyEmpty(game structures.Game) bool {
	return len(game.Players) < constants.MaxPlayers
}
func joinPlayerIntoGame(client net.Conn, message string) {

	lobbyName := message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:]
	gameMapMutex.Lock()
	if game, ok := gameMap[lobbyName]; ok {
		if isLobbyEmpty(game) {
			if _, exists := clientsMap[client]; exists {
				playerID := len(game.Players) + 1
				game.Players[playerID] = clientsMap[client]
				fmt.Printf("User %s joined lobby %s\n", clientsMap[client].Nickname, lobbyName)
				delete(clientsMap, client)

			} else {
				fmt.Println("User not found in clientsMap.")
			}
		} else {
			fmt.Println("Lobby is not empty.")
		}
	} else {
		fmt.Printf("Lobby %s not found in gameMap.\n", lobbyName)
	}
	gameMapMutex.Unlock()
}

func createNickForConnection(client net.Conn, message string) bool {
	messageType := message[len(constants.MessageHeader)+constants.MessageLengthFormat : len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength]
	if messageType == "nick" {
		clientsMap[client] = structures.Player{
			Nickname: message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:],
			Socket:   client,
		}
		return true
	} else {
		return false
	}
}

func isLengthValid(message string) bool {

	if len(message) < (len(constants.MessageHeader) + constants.MessageTypeLength + constants.MessageLengthFormat) {
		return false
	}
	// Magic
	magic := message[:len(constants.MessageHeader)]

	if magic != constants.MessageHeader {
		fmt.Printf("Magic:%s, Constant:%s\n", magic, constants.MessageHeader)
		return false
	}
	// Message Length
	lengthStr := message[len(constants.MessageHeader) : len(constants.MessageHeader)+constants.MessageLengthFormat]
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return false
	}

	// is message length valid?
	if length != len(message)-len(constants.MessageHeader)-constants.MessageLengthFormat-constants.MessageTypeLength {
		fmt.Printf("LengthFromMessage:%d, CalculatedLength:%s\n", length, len(message)-len(constants.MessageHeader)-constants.MessageLengthFormat-constants.MessageTypeLength)
		return false
	}

	// Extract the type part (next 4 characters)
	messageType := message[len(constants.MessageHeader)+constants.MessageLengthFormat : len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength]

	messageValue := message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:]

	fmt.Printf("Magic: %s\n", magic)
	fmt.Printf("Length: %d\n", length)
	fmt.Printf("Type: %s\n", messageType)
	fmt.Printf("Message: %s\n", messageValue)

	return true
}
