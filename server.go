package main

import (
	"UPS_sem/constants"
	"UPS_sem/structures"
	"UPS_sem/utils"
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"sort"
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

var (
	clientsMap        = make(map[net.Conn]structures.Player)
	gameMap           = make(map[string]structures.Game)
	dictionary        []structures.DictionaryItem
	dictionaryMutex   sync.Mutex
	clientsMapMutex   sync.Mutex
	gameMapMutex      sync.Mutex
	letterPoints      = constants.LetterPoints()
	letterPointsMutex sync.Mutex
)

func main() {
	initializeGameMap()
	createDictionary()

	go pingRoutine()

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

func pingRoutine() {
	for {
		pingAllClients()
		time.Sleep(5 * time.Second)
	}
}

func pingAllClients() {
	clientsMapMutex.Lock()
	message := utils.CreatePingMessage()
	for i, player := range clientsMap {
		if player.PingCounter > 0 && player.PingCounter < 10 {
			fmt.Printf("Hrac %s ma problem s connectionem.", player.Nickname)
		} else {
			fmt.Printf("Hrac %s je v cajku.", player.Nickname)
		}
		player.Socket.Write([]byte(message))
		player.PingCounter++
		if player.PingCounter <= 10 {
			clientsMap[i] = player
		} else {
			fmt.Println("Odpojuju kokota: ", player.Nickname)
			delete(clientsMap, player.Socket)
			//updateLobbyInfoInOtherClients()
		}
	}
	clientsMapMutex.Unlock()

	gameMapMutex.Lock()
	for gameID, game := range gameMap {
		for playerID, player := range game.Players {
			if player.PingCounter > 0 && player.PingCounter < 10 {
				fmt.Printf("Hrac %s ma problem s connectionem.", player.Nickname)
			} else {
				fmt.Printf("Hrac %s je v cajku.", player.Nickname)
			}
			player.Socket.Write([]byte(message))
			player.PingCounter++

			if player.PingCounter <= 10 {
				gameMap[gameID].Players[playerID] = player
			} else {
				fmt.Println("Disconnecting player: ", player.Nickname)
				delete(gameMap[gameID].Players, playerID)
				sendMessageToCancelGame(game)
				clientsMapMutex.Lock()
				movePlayersBackToMainLobby(&game)
				clientsMapMutex.Unlock()

				game.GameData.IsLobby = true
				gameMap[gameID] = game
				updateLobbyInfoInOtherClients()
			}
		}
	}

	gameMapMutex.Unlock()

}

func sendMessageToCancelGame(game structures.Game) {
	message := utils.CreateCancelMessage()
	for _, player := range game.Players {
		player.Socket.Write([]byte(message))
	}
}

func createDictionary() {
	content, _ := ioutil.ReadFile("dictionary/" + constants.DictionaryFile)

	sentenceArray := strings.Split(string(content), "\n")

	var cleanedSentences []structures.DictionaryItem
	for _, sentence := range sentenceArray {
		if sentence != "" {
			parts := strings.Split(sentence, ";")
			if len(parts) == 2 {
				item := structures.DictionaryItem{
					Sentence: parts[0],
					Hint:     parts[1],
				}
				cleanedSentences = append(cleanedSentences, item)
			}
		}
	}

	dictionaryMutex.Lock()
	dictionary = cleanedSentences
	dictionaryMutex.Unlock()
}

func initializeGameMap() {
	gameMapMutex.Lock()
	for i := 1; i <= constants.RoomsCount; i++ {
		lobbyID := fmt.Sprintf("lobby%d", i)
		gameMap[lobbyID] = structures.Game{
			ID:      lobbyID,
			Players: make(map[int]structures.Player),
			GameData: structures.GameState{
				IsLobby: true,
			},
		}
	}
	gameMapMutex.Unlock()
}

func handleConnection(client net.Conn) {
	//defer client.Close()

	reader := bufio.NewReader(client)

	for {
		readBuffer, err := reader.ReadBytes('\n')

		if err != nil {
			clientsMapMutex.Lock()
			//fmt.Println("Zabijim: ", clientsMap[client].Nickname)
			clientsMapMutex.Unlock()
			//fmt.Println("Client disconnected.:", client)
			return
		}
		// Convert to string and remove trailing newline characters
		message := strings.TrimRight(string(readBuffer), "\r\n")
		fmt.Println("Msg:", message)

		if utils.IsLengthValid(message) {
			fmt.Println("Message structure is valid.")
			handleMessage(message, client)
		} else {
			fmt.Println("Message structure is invalid. Closing connection.")
			return
		}
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
			sendLobbyInfo(client)
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
		case "play":
			startTheGame(client, extractedMessage)
		case "lett":
			receiveLetter(client, extractedMessage)
		case "pong":
			//handlePongMessage(client)
		default:
			fmt.Println("Unknown command ", messageType)
		}
		resetPlayerCounter(client)
	}
	clientsMapMutex.Unlock()
}

func handlePongMessage(client net.Conn) {
	resetPlayerCounter(client)
}

func resetPlayerCounter(client net.Conn) {
	for _, player := range clientsMap {
		if player.Socket == client {
			player.PingCounter = 0
			clientsMap[client] = player
			return
		}
	}

	gameMapMutex.Lock()
	for _, game := range gameMap {
		for i, player := range game.Players {
			if player.Socket == client {
				player.PingCounter = 0
				gameMap[game.ID].Players[i] = player
				gameMapMutex.Unlock()
				return
			}
		}

	}
	gameMapMutex.Unlock()

}

func sendLobbyInfo(client net.Conn) {
	gameMapMutex.Lock()
	var gameStrings []string
	for _, game := range gameMap {
		playerCount := len(game.Players)
		isLobby := 0
		if game.GameData.IsLobby {
			isLobby = 1
		}

		gameString := fmt.Sprintf("%s|%d|%d|%d", game.ID, constants.MaxPlayers, playerCount, isLobby)
		gameStrings = append(gameStrings, gameString)
	}
	sort.Strings(gameStrings)

	gameMapMutex.Unlock()
	finalMessage := utils.CreateLobbyInfoMessage(gameStrings)
	gameMapMutex.Lock()
	client.Write([]byte(finalMessage))
	gameMapMutex.Unlock()
}

func receiveLetter(client net.Conn, message string) {
	player := findPlayerBySocketReturn(client)
	if len(message) != 1 {
		fmt.Println("Nevalidni zprava more")
		return
	}

	if player == nil {
		fmt.Println("Nenasel jsem daneho hrace, koncim")
		return
	}

	lobbyID := findLobbyWithPlayer(*player).ID
	lobby, ok := gameMap[lobbyID]
	if ok {
		gameMapMutex.Lock()
		lobby.GameData.PlayersPlayed[*player] = true
		playerMadeMove(&lobby, *player, message)
		gameMap[lobbyID] = lobby
		if lobby.GameData.IsLobby {
			print("Hra asi skoncila, posilam nove informace typkum")
			updateLobbyInfoInOtherClients()
		}
		gameMapMutex.Unlock()
	}

}

func startNewRound(game *structures.Game) {
	dictionaryItem := selectRandomSentence()
	game.GameData.SentenceToGuess = dictionaryItem.Sentence
	game.GameData.Hint = dictionaryItem.Hint
	game.GameData.CharactersSelected = []string{}
	game.GameData.PlayerLetters = make(map[structures.Player]string)

	for _, player := range game.Players {
		game.GameData.PlayersPlayed[player] = false
	}
}

func didGameEnded(game *structures.Game) bool {
	gameData := game.GameData

	for _, points := range gameData.PlayerPoints {
		if points >= constants.PointsNeededToWin {
			return true
		}
	}

	return false
}
func areAllPlayersPlayed(playersPlayed map[structures.Player]bool) bool {
	for _, played := range playersPlayed {
		if !played {
			return false
		}
	}
	return true
}

func playerMadeMove(game *structures.Game, player structures.Player, letter string) {
	game.GameData.PlayersPlayed[player] = true
	game.GameData.PlayerLetters[player] = letter

	if areAllPlayersPlayed(game.GameData.PlayersPlayed) {
		completeSentenceWithLetters(game)
		if didGameEnded(game) {
			gameEndedMessage(game)
			movePlayersBackToMainLobby(game)
			game.GameData.IsLobby = true
		} else {
			if isSentenceGuessed(game) {
				sendSentenceGuessedMessage(game)
				printPlayerPoints(game.GameData.PlayerPoints)
				initializeNextRound(game)
				startNewRound(game)
			}
			messageToClients := utils.GameStartedWithInitInfo(*game)
			for _, player := range gameMap[game.ID].Players {
				player.Socket.Write([]byte(messageToClients))
				game.GameData.PlayersPlayed[player] = false
			}
		}
	}
}

func gameEndedMessage(game *structures.Game) {
	message := utils.CreateGameEndingMessage(game)
	for _, player := range game.Players {
		player.Socket.Write([]byte(message))
	}
}

func sendSentenceGuessedMessage(game *structures.Game) {
	message := utils.CreateSentenceGuessedMessage(game)
	for _, player := range game.Players {
		player.Socket.Write([]byte(message))
	}
}

func initializeNextRound(game *structures.Game) {
	for _, player := range game.Players {
		game.GameData.PlayersPlayed[player] = false
	}
	game.GameData.PlayerLetters = make(map[structures.Player]string)
}

func completeSentenceWithLetters(game *structures.Game) {
	for _, player := range game.Players {
		calculatePoints(&player, game)
	}
}

func calculatePoints(player *structures.Player, game *structures.Game) {
	letter := game.GameData.PlayerLetters[*player]
	result := calculatePointPerLetter(letter, game.GameData.SentenceToGuess)
	game.GameData.PlayerPoints[*player] += result

	if contains(game.GameData.CharactersSelected, letter) {
		fmt.Println("Uz obsahuje, nic nepridavam, je to prvek: ", letter)
	} else {
		fmt.Println("Pridavam novy prvek, ktery tam jeste nebyl je jim: ", letter)
		game.GameData.CharactersSelected = append(game.GameData.CharactersSelected, letter)
	}
}

func calculatePointPerLetter(character, sentence string) int {
	characterLower := strings.ToLower(character)
	sentenceLower := strings.ToLower(sentence)

	count := 0
	for _, char := range sentenceLower {
		if string(char) == characterLower {
			count++
		}
	}

	letterPointsMutex.Lock()
	defer letterPointsMutex.Unlock()

	return letterPoints[characterLower] * count

}

func contains(slice []string, element string) bool {
	for _, el := range slice {
		if el == element {
			return true
		}
	}
	return false
}

func movePlayersBackToMainLobby(game *structures.Game) {
	for _, player := range game.Players {
		clientsMap[player.Socket] = player
	}

	game.Players = make(map[int]structures.Player)
}

func isSentenceGuessed(lobby *structures.Game) bool {
	sentenceToGuess := strings.ToLower(lobby.GameData.SentenceToGuess)
	charactersSelected := strings.ToLower(strings.Join(lobby.GameData.CharactersSelected, ""))
	fmt.Println("novy kolo: ", sentenceToGuess, charactersSelected)
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

func startTheGame(client net.Conn, message string) {
	player := findPlayerBySocketReturn(client)
	if player == nil {
		fmt.Println("Nenasel jsem daneho hrace v zadnem lobby, koncim")
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
func initializePlayerPoints(gameData *structures.GameState, players map[int]structures.Player) {
	for _, player := range players {
		gameData.PlayerPoints[player] = 0
	}
}

func printPlayerPoints(playerPoints map[structures.Player]int) {
	for player, points := range playerPoints {
		fmt.Printf("Player %v has %d points\n", player, points)
	}
}

func switchLobbyToGame(lobbyID string) {
	gameMapMutex.Lock()
	defer gameMapMutex.Unlock()

	if existingGame, ok := gameMap[lobbyID]; ok {
		existingGame.GameData.IsLobby = false
		dictionaryItem := selectRandomSentence()
		existingGame.GameData.SentenceToGuess = dictionaryItem.Sentence
		existingGame.GameData.Hint = dictionaryItem.Hint
		existingGame.GameData.CharactersSelected = []string{}
		existingGame.GameData.PlayerPoints = make(map[structures.Player]int)
		existingGame.GameData.PlayersPlayed = make(map[structures.Player]bool)
		existingGame.GameData.PlayerLetters = make(map[structures.Player]string)
		initializePlayerPoints(&existingGame.GameData, existingGame.Players)
		printPlayerPoints(existingGame.GameData.PlayerPoints)

		gameMap[lobbyID] = existingGame
		messageToClients := utils.GameStartedWithInitInfo(existingGame)
		for _, player := range gameMap[lobbyID].Players {
			player.Socket.Write([]byte(messageToClients))
			existingGame.GameData.PlayersPlayed[player] = false
		}

		return
	}
}

func selectRandomSentence() structures.DictionaryItem {
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
				playerMovedToGameLobby(game.Players[playerID])
				updateLobbyInfoInOtherClients()
				sendInfoAboutStart(game)
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

func updateLobbyInfoInOtherClients() {
	for _, player := range clientsMap {
		gameMapMutex.Unlock()
		sendLobbyInfo(player.Socket)
		gameMapMutex.Lock()
	}
}

func playerMovedToGameLobby(player structures.Player) {
	player.Socket.Write([]byte(utils.LobbyJoined(true)))

}

func sendInfoAboutStart(game structures.Game) {
	for _, player := range game.Players {
		gameMapMutex.Unlock()
		player.Socket.Write([]byte(utils.CanBeStarted(canLobbyBeStarted(game), len(game.Players), constants.MaxPlayers)))
		gameMapMutex.Lock()
	}
}

func createNickForConnection(client net.Conn, message string) bool {
	messageType := message[len(constants.MessageHeader)+constants.MessageLengthFormat : len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength]
	if messageType == "nick" {
		clientsMap[client] = structures.Player{
			Nickname:    message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:],
			Socket:      client,
			PingCounter: 0,
		}
		return true
	} else {
		return false
	}
}
