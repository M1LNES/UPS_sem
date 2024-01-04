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

/* Constants imported from config.go */
const (
	connHost = constants.ConnHost
	connPort = constants.ConnPort
	connType = constants.ConnType
)

/* Global variables for storing data */
var (
	mainLobbyMap          = make(map[net.Conn]structures.Player)
	gamingLobbiesMap      = make(map[string]structures.Game)
	dictionary            []structures.DictionaryItem
	dictionaryMutex       sync.Mutex
	mainLobbyMapMutex     sync.Mutex
	gamingLobbiesMapMutex sync.Mutex
	letterPoints          = constants.LetterPoints()
	letterPointsMutex     sync.Mutex
)

/* Main function that executes the server */
func main() {

	if !utils.ValidateConfig() {
		fmt.Println("Config File is not valid. Closing server!")
		return
	}

	initializegamingLobbiesMap()
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

		//fmt.Println("Client " + client.RemoteAddr().String() + " connected.")

		go handleConnection(client)
	}
}

/* function that every 5 seconds call method for pinging clients*/
func pingRoutine() {
	for {
		pingAllClients()
		time.Sleep(5 * time.Second)
	}
}

/* Function that ping clients and manage their states - removing them from lobbies etc. */
func pingAllClients() {
	mainLobbyMapMutex.Lock()
	gamingLobbiesMapMutex.Lock()

	message := utils.CreatePingMessage()
	for i, player := range mainLobbyMap {
		player.Socket.Write([]byte(message))
		player.PingCounter++
		if player.PingCounter <= 12 {
			mainLobbyMap[i] = player
		} else {
			player.Socket.Close()
			delete(mainLobbyMap, player.Socket)
		}
	}

	for _, game := range gamingLobbiesMap {
		for _, player := range game.Players {
			if player.PingCounter > 0 && player.PingCounter < 12 {
				utils.SendInfoAboutPendingUser(game, player)
				//fmt.Printf("Hrac %s ma problem s connectionem.", player.Nickname)
			} else if player.PingCounter == 0 {
				utils.SendInfoAboutConnectedUser(game, player)
				//fmt.Printf("Hrac %s je v cajku.", player.Nickname)
			} else {
				//fmt.Printf("Hrac %s bude odpojen", player.Nickname)

			}
			player.Socket.Write([]byte(message))
			player.PingCounter++
			if player.PingCounter <= 12 {
				gamingLobbiesMap[game.ID].Players[player.Nickname] = player
			} else {
				fmt.Println("Disconnecting player: ", player.Nickname)
				gamingLobbiesMap[game.ID].Players[player.Nickname].Socket.Close()
				delete(gamingLobbiesMap[game.ID].Players, player.Nickname)
				sendMessageToCancelGame(game)
				movePlayersBackToMainLobby(&game)

				game.GameData.IsLobby = true
				gamingLobbiesMap[game.ID] = game
				updateLobbyInfoInOtherClients()
			}
		}
	}

	gamingLobbiesMapMutex.Unlock()
	mainLobbyMapMutex.Unlock()
}

/* Function that sends message to players that game was canceled */
func sendMessageToCancelGame(game structures.Game) {
	message := utils.CreateCancelMessage()
	for _, player := range game.Players {
		player.Socket.Write([]byte(message))
	}
}

/* Function that creates dictionary for the letters */
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

/* Function that create and initialize gaming lobbies */
func initializegamingLobbiesMap() {
	gamingLobbiesMapMutex.Lock()
	for i := 1; i <= constants.RoomsCount; i++ {
		lobbyID := fmt.Sprintf("lobby%d", i)
		gamingLobbiesMap[lobbyID] = structures.Game{
			ID:      lobbyID,
			Players: make(map[string]structures.Player),
			GameData: structures.GameState{
				IsLobby: true,
			},
		}
	}
	gamingLobbiesMapMutex.Unlock()
}

/* Function that handle connection for each client */
func handleConnection(client net.Conn) {
	defer client.Close()
	reader := bufio.NewReader(client)

	for {
		readBuffer, err := reader.ReadBytes('\n')

		if err != nil {
			client.Close()
			return
		}
		message := strings.TrimRight(string(readBuffer), "\r\n")

		if utils.IsLengthValid(message) {
			handleMessage(message, client)
		} else {
			sendErrorAndDisconnect(client, "Message structure is not valid!")
			return
		}
	}
}

/* Function that finds if player with socket exists in gaming lobbies */
func findPlayerBySocket(client net.Conn) bool {

	for _, gameState := range gamingLobbiesMap {
		for _, player := range gameState.Players {
			if player.Socket == client {
				return true
			}
		}
	}
	return false
}

/* Function that finds player by his socket in gaming lobbies (not in main lobby) */
func findPlayerBySocketReturn(client net.Conn) *structures.Player {
	for _, gameState := range gamingLobbiesMap {
		for _, player := range gameState.Players {
			if player.Socket == client {
				return &player
			}
		}
	}
	return nil
}

/* Function that handle messages from client */
func handleMessage(message string, client net.Conn) {
	gamingLobbiesMapMutex.Lock()
	mainLobbyMapMutex.Lock()

	messageType := message[len(constants.MessageHeader)+constants.MessageLengthFormat : len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength]
	extractedMessage := message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:]

	if messageType == "nick" && playerNickInGameWithDifferentSocket(extractedMessage, client) {
		renewStateToPlayer(extractedMessage, client)
	} else if _, exists := mainLobbyMap[client]; !exists && findPlayerBySocket(client) == false {
		if createNickForConnection(client, message) {
			sendLobbyInfo(client)
			//printGamingLobbiesMap()
		} else {
			sendErrorAndDisconnect(client, "Firstly you must identify yourself, aborting!")
		}
	} else {

		switch messageType {
		case "join":
			joinPlayerIntoGame(client, message)
		case "play":
			startTheGame(client, extractedMessage)
		case "lett":
			receiveLetter(client, extractedMessage, message)
		case "pong":
			//handlePongMessage(client)
		case "retr":
			//resendClientInfo(client, message)
		case constants.Error:
			client.Close()
		default:
			sendErrorAndDisconnect(client, "Unknown command, aborting!")
		}
		resetPlayerCounter(client)
	}
	gamingLobbiesMapMutex.Unlock()
	mainLobbyMapMutex.Unlock()
}

/* Function that disconnect user due to his invalid message format */
func sendErrorAndDisconnect(client net.Conn, s string) {
	errorMess := utils.CreateErrorMessage(s)
	client.Write([]byte(errorMess))
	client.Close()
}

/* Function that renew state to short-disconnected user */
func renewStateToPlayer(message string, client net.Conn) {
	game, player := playerNickInGameWithDifferentSocketReturn(message, client)
	player.Socket.Close()
	player.Socket = client
	player.PingCounter = 0
	game.Players[player.Nickname] = *player
	gamingLobbiesMap[game.ID] = *game
	player.Socket.Write([]byte(utils.LobbyJoined(true)))

	sendInfoAboutStartToClient(*player, *game)
	if !gamingLobbiesMap[game.ID].GameData.IsLobby {
		//fmt.Println("Pry to neni lobby")
		resendClientInfo(client)
	} else {
		//fmt.Println("je to lobby")
	}
}

/* Function that sends info to client if the game can be started */
func sendInfoAboutStartToClient(player structures.Player, game structures.Game) {
	player.Socket.Write([]byte(utils.CanBeStarted(canLobbyBeStarted(game), len(game.Players), constants.MaxPlayers)))
}

/* Function that returns gaming lobby and client with specified nick */
func playerNickInGameWithDifferentSocketReturn(nick string, socket net.Conn) (*structures.Game, *structures.Player) {
	for _, game := range gamingLobbiesMap {
		for _, player := range game.Players {
			if player.Nickname == nick && player.Socket != socket {
				return &game, &player
			}
		}
	}
	return nil, nil
}

/* Function that finds if player with name "nick" exists in gaming lobbies */
func playerNickInGameWithDifferentSocket(nick string, socket net.Conn) bool {
	for _, game := range gamingLobbiesMap {
		for _, player := range game.Players {
			if player.Nickname == nick {
				return player.Socket != socket
			}
		}
	}
	return false
}

/* Function that resends info to client */
func resendClientInfo(client net.Conn) {
	player := findPlayerBySocketReturn(client)
	lobbyID := findLobbyWithPlayer(*player).ID
	lobby, _ := gamingLobbiesMap[lobbyID]

	messageFinal := utils.CreateResendStateMessage(&lobby, *player)
	client.Write([]byte(messageFinal))
}

/* Function that was reseting player counter  */
func handlePongMessage(client net.Conn) {
	resetPlayerCounter(client)
}

/* Function that reset counter of player with specified socket */
func resetPlayerCounter(client net.Conn) {
	for _, player := range mainLobbyMap {
		if player.Socket == client {
			player.PingCounter = 0
			mainLobbyMap[client] = player
			return
		}
	}

	for _, game := range gamingLobbiesMap {
		for i, player := range game.Players {
			if player.Socket == client {
				player.PingCounter = 0
				gamingLobbiesMap[game.ID].Players[i] = player
				return
			}
		}

	}

}

/* Function that sends info about lobby */
func sendLobbyInfo(client net.Conn) {
	var gameStrings []string
	for _, game := range gamingLobbiesMap {
		playerCount := len(game.Players)
		isLobby := 0
		if game.GameData.IsLobby {
			isLobby = 1
		}

		gameString := fmt.Sprintf("%s|%d|%d|%d", game.ID, constants.MaxPlayers, playerCount, isLobby)
		gameStrings = append(gameStrings, gameString)
	}
	sort.Strings(gameStrings)

	finalMessage := utils.CreateLobbyInfoMessage(gameStrings)
	client.Write([]byte(finalMessage))
}

/* Function that handle letter message and is also solving gaming logic */
func receiveLetter(client net.Conn, message string, wholeMessage string) {
	player := findPlayerBySocketReturn(client)

	if !utils.IsCharacterValid(message) {
		sendErrorAndDisconnect(client, "Invalid letter. Aborting.")
		return
	}

	if player == nil {
		sendErrorAndDisconnect(client, "Invalid player! Aborting.")
		return
	}

	if findLobbyWithPlayer(*player) == nil {
		sendErrorAndDisconnect(client, "Player is not in any gaming lobby.")
		return
	}

	lobbyID := findLobbyWithPlayer(*player).ID

	lobby, ok := gamingLobbiesMap[lobbyID]
	if ok && !contains(gamingLobbiesMap[lobbyID].GameData.CharactersSelected, message) {
		if gamingLobbiesMap[lobbyID].GameData.PlayersPlayed[player.Nickname] {
			infoMess := utils.CreateInfoMessage("This user already played.")
			client.Write([]byte(infoMess))
			return
		}

		player.Socket.Write([]byte(wholeMessage + "\n"))
		playerMadeMove(&lobby, *player, message)
		gamingLobbiesMap[lobbyID] = lobby
		if lobby.GameData.IsLobby {
			updateLobbyInfoInOtherClients()
		}
	} else {
		sendErrorAndDisconnect(client, "This letter was already selected rounds before.")
		// TODO test
	}
}

/* Function that initialize new state of game */
func startNewRound(game *structures.Game) {
	dictionaryItem := selectRandomSentence()
	game.GameData.SentenceToGuess = dictionaryItem.Sentence
	game.GameData.Hint = dictionaryItem.Hint
	game.GameData.CharactersSelected = []string{}
	game.GameData.PlayerLetters = make(map[string]string)

	for _, player := range game.Players {
		game.GameData.PlayersPlayed[player.Nickname] = false
	}
}

/* Function that returns if game ended */
func didGameEnded(game *structures.Game) bool {
	gameData := game.GameData

	for _, points := range gameData.PlayerPoints {
		if points >= constants.PointsNeededToWin {
			return true
		}
	}

	return false
}

/* Function that goes through map and find if all players played */
func areAllPlayersPlayed(playersPlayed map[string]bool) bool {
	for _, played := range playersPlayed {
		if !played {
			return false
		}
	}
	return true
}

/* Function that handles logic of players turn */
func playerMadeMove(game *structures.Game, player structures.Player, letter string) {
	game.GameData.PlayersPlayed[player.Nickname] = true
	game.GameData.PlayerLetters[player.Nickname] = letter

	if areAllPlayersPlayed(game.GameData.PlayersPlayed) {
		completeSentenceWithLetters(game)
		//fmt.Println("Doplnil jsem vetu pismenky :)!")

		if didGameEnded(game) {
			gameEndedMessage(game)
			movePlayersBackToMainLobby(game)
			game.GameData.IsLobby = true
			gamingLobbiesMap[game.ID] = *game
		} else {
			if isSentenceGuessed(game) {
				sendSentenceGuessedMessage(game)
				//printPlayerPoints(game.GameData.PlayerPoints)
				initializeNextRound(game)
				startNewRound(game)
			}
			messageToClients := utils.GameStartedWithInitInfo(*game)
			for _, player := range gamingLobbiesMap[game.ID].Players {
				player.Socket.Write([]byte(messageToClients))
				game.GameData.PlayersPlayed[player.Nickname] = false
			}
		}
	} else {
		//fmt.Println("Jeste nehrali vsichni!")
	}
}

/* Function that send message to client that game ended */
func gameEndedMessage(game *structures.Game) {
	message := utils.CreateGameEndingMessage(game)
	for _, player := range game.Players {
		player.Socket.Write([]byte(message))
	}
}

/* Function send message that sentence was guessed */
func sendSentenceGuessedMessage(game *structures.Game) {
	message := utils.CreateSentenceGuessedMessage(game)
	for _, player := range game.Players {
		player.Socket.Write([]byte(message))
	}
}

/* Function that initialize new round */
func initializeNextRound(game *structures.Game) {
	for _, player := range game.Players {
		game.GameData.PlayersPlayed[player.Nickname] = false
	}
	game.GameData.PlayerLetters = make(map[string]string)
}

/* Function that complete sentence with new letters from clients */
func completeSentenceWithLetters(game *structures.Game) {
	for _, player := range game.Players {
		calculatePoints(&player, game)
	}
}

/* Function that calculates points for players */
func calculatePoints(player *structures.Player, game *structures.Game) {
	letter := game.GameData.PlayerLetters[player.Nickname]
	result := calculatePointPerLetter(letter, game.GameData.SentenceToGuess)
	game.GameData.PlayerPoints[player.Nickname] += result

	if contains(game.GameData.CharactersSelected, letter) {
		//fmt.Println("Uz obsahuje, nic nepridavam, je to prvek: ", letter)
	} else {
		//fmt.Println("Pridavam novy prvek, ktery tam jeste nebyl je jim: ", letter)
		game.GameData.CharactersSelected = append(game.GameData.CharactersSelected, letter)
	}
}

/* Function that calculate point for players turn */
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

/* Function that finds if array already contains a character*/
func contains(slice []string, element string) bool {
	for _, el := range slice {
		if el == element {
			return true
		}
	}
	return false
}

/* Function that move player from gaming lobby to main lobby */
func movePlayersBackToMainLobby(game *structures.Game) {
	for _, player := range game.Players {
		mainLobbyMap[player.Socket] = player
		delete(game.Players, player.Nickname)
	}

	//game.Players = make(map[string]structures.Player)
}

/* Function that handle if sentence is guessed */
func isSentenceGuessed(lobby *structures.Game) bool {
	sentenceToGuess := strings.ToLower(lobby.GameData.SentenceToGuess)
	charactersSelected := strings.ToLower(strings.Join(lobby.GameData.CharactersSelected, ""))
	//fmt.Println("novy kolo: ", sentenceToGuess, charactersSelected)
	//fmt.Printf("Characters selected %s Sentence %s \n", charactersSelected, sentenceToGuess)
	for _, char := range sentenceToGuess {
		if !unicode.IsLetter(char) {
			continue
		}

		if !strings.ContainsRune(charactersSelected, unicode.ToLower(char)) {
			return false
		}
	}
	return true
}

/* Function start the game */
func startTheGame(client net.Conn, message string) {
	player := findPlayerBySocketReturn(client)
	if player == nil {
		sendErrorAndDisconnect(client, "Player is not in any gaming lobby. Aborting.")
		return
	}
	lobby := findLobbyWithPlayer(*player)
	if canLobbyBeStarted(*lobby) {
		switchLobbyToGame(lobby.ID)
		updateLobbyInfoInOtherClients()
	} else {
		sendErrorAndDisconnect(client, "Lobby is not ready to play! Aborting.")
		return
	}
}

/* Function that return if lobby can be started */
func canLobbyBeStarted(lobby structures.Game) bool {
	return len(lobby.Players) > 1 && lobby.GameData.IsLobby
}

/* Function that return gaming lobby with player */
func findLobbyWithPlayer(player structures.Player) *structures.Game {
	for _, game := range gamingLobbiesMap {
		for _, p := range game.Players {
			if p == player {
				return &game
			}
		}
	}
	return nil
}

/* Function for initializing player points */
func initializePlayerPoints(gameData *structures.GameState, players map[string]structures.Player) {
	for _, player := range players {
		gameData.PlayerPoints[player.Nickname] = 0
	}
}

/* Debugging function */
func printPlayerPoints(playerPoints map[string]int) {
	for player, points := range playerPoints {
		fmt.Printf("Player %v has %d points\n", player, points)
	}
}

/* Function that switch lobby to game*/
func switchLobbyToGame(lobbyID string) {
	if selectedGame, ok := gamingLobbiesMap[lobbyID]; ok {
		selectedGame.GameData.IsLobby = false
		dictionaryItem := selectRandomSentence()
		selectedGame.GameData.SentenceToGuess = dictionaryItem.Sentence
		selectedGame.GameData.Hint = dictionaryItem.Hint
		selectedGame.GameData.CharactersSelected = []string{}
		selectedGame.GameData.PlayerPoints = make(map[string]int)
		selectedGame.GameData.PlayersPlayed = make(map[string]bool)
		selectedGame.GameData.PlayerLetters = make(map[string]string)
		initializePlayerPoints(&selectedGame.GameData, selectedGame.Players)
		//printPlayerPoints(selectedGame.GameData.PlayerPoints)

		gamingLobbiesMap[lobbyID] = selectedGame
		messageToClients := utils.GameStartedWithInitInfo(selectedGame)
		for _, player := range gamingLobbiesMap[lobbyID].Players {
			player.Socket.Write([]byte(messageToClients))
			selectedGame.GameData.PlayersPlayed[player.Nickname] = false
		}

		return
	}
}

/* Function that return one of the dictionary item from map */
func selectRandomSentence() structures.DictionaryItem {
	rand.Seed(time.Now().UnixNano())
	dictionaryMutex.Lock()
	index := rand.Intn(len(dictionary))
	dictionaryMutex.Unlock()
	return dictionary[index]
}

/* Function that returns if user can join lobby or not */
func isLobbyEmpty(game structures.Game) bool {
	return len(game.Players) < constants.MaxPlayers
}

/* Function that handles joining user into game */
func joinPlayerIntoGame(client net.Conn, message string) {
	lobbyName := message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:]
	//printGamingLobbiesMap()
	if game, ok := gamingLobbiesMap[lobbyName]; ok {
		if isLobbyEmpty(game) && game.GameData.IsLobby {
			if _, exists := mainLobbyMap[client]; exists {
				nick := mainLobbyMap[client].Nickname
				game.Players[mainLobbyMap[client].Nickname] = mainLobbyMap[client]
				//fmt.Printf("User %s joined lobby %s\n", mainLobbyMap[client].Nickname, lobbyName)
				delete(mainLobbyMap, client)
				playerMovedToGameLobby(game.Players[nick])
				updateLobbyInfoInOtherClients()
				sendInfoAboutStart(game)
			} else {
				sendErrorAndDisconnect(client, "User not found in mainMapLobby")
			}
		} else {
			messageToClient := utils.LobbyCannotBeJoined()
			client.Write([]byte(messageToClient))
		}
	} else {
		sendErrorAndDisconnect(client, "Lobby not found in gamingLobbiesMap")
	}
	//printGamingLobbiesMap()
}

/* Function that update info about gaming lobbies in all clients in main lobby */
func updateLobbyInfoInOtherClients() {
	for _, player := range mainLobbyMap {
		sendLobbyInfo(player.Socket)
	}
}

/* Function that send client message that he was moved to main lobby */
func playerMovedToGameLobby(player structures.Player) {
	player.Socket.Write([]byte(utils.LobbyJoined(true)))
}

/* Function that send info about start */
func sendInfoAboutStart(game structures.Game) {
	for _, player := range game.Players {
		player.Socket.Write([]byte(utils.CanBeStarted(canLobbyBeStarted(game), len(game.Players), constants.MaxPlayers)))
	}
}

/* Function that create player instance */
func createNickForConnection(client net.Conn, message string) bool {
	messageType := message[len(constants.MessageHeader)+constants.MessageLengthFormat : len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength]
	if messageType == "nick" {
		nickname := message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:]

		for existingClient, existingPlayer := range mainLobbyMap {
			if existingPlayer.Nickname == nickname {
				delete(mainLobbyMap, existingClient)
				break
			}
		}

		mainLobbyMap[client] = structures.Player{
			Nickname:    nickname,
			Socket:      client,
			PingCounter: 0,
		}
		return true
	} else {
		return false
	}
}

/* Function for debugging */
func printGamingLobbiesMap() {
	fmt.Println("Printing gamingLobbiesMap:")
	for lobbyID, game := range gamingLobbiesMap {
		fmt.Printf("Lobby ID: %s\n", lobbyID)
		PrintPlayersInLobby(&game)
		fmt.Println("--------------------")
	}
}

/* Function for debugging */
func PrintPlayersInLobby(g *structures.Game) {
	fmt.Printf("Players in Lobby (Game ID: %s):\n", g.ID)

	if len(g.Players) == 0 {
		fmt.Println("No players in the lobby.")
		return
	}

	for _, player := range g.Players {
		fmt.Printf("Player ID: %s\n", player.Nickname)
	}
}
