package utils

import (
	"UPS_sem/constants"
	"UPS_sem/structures"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// SENDING

// RECEIVING

func LobbyJoined(success bool) string {
	magic := constants.MessageHeader
	messageType := constants.LobbyJoining
	successStr := "0"
	if success {
		successStr = "1"
	}

	message := fmt.Sprintf("%s%03d%s%s\n", magic, len(successStr), messageType, successStr)
	return message
}

func CanBeStarted(canBeStarted bool, currentPlayers int, maxPlayers int) string {
	magic := constants.MessageHeader
	messageType := constants.CanGameStart
	canBeStartedStr := "0"
	if canBeStarted {
		canBeStartedStr = "1"
	}

	messageBody := fmt.Sprintf("%s|%d|%d", canBeStartedStr, currentPlayers, maxPlayers)
	message := fmt.Sprintf("%s%03d%s%s\n", magic, len(messageBody), messageType, messageBody)
	return message
}

func GameStartedWithInitInfo(game structures.Game) string {
	magic := constants.MessageHeader
	messageType := constants.GameStartedInit
	players := getPlayerNicknamesWithPoints(game)
	charactersSelectedSoFar := selectedCharactersFromGame(game)
	hint := game.GameData.Hint
	maskedSentence := maskSentence(game.GameData.CharactersSelected, game.GameData.SentenceToGuess)
	messageBody := fmt.Sprintf("%s|%s|%s|%s", players, charactersSelectedSoFar, maskedSentence, hint)
	finalMessage := fmt.Sprintf("%s%03d%s%s\n", magic, len(messageBody), messageType, messageBody)
	return finalMessage
}

func selectedCharactersFromGame(game structures.Game) string {
	return strings.Join(game.GameData.CharactersSelected, "")
}

func getPlayerNicknamesWithPoints(game structures.Game) string {
	var nicknames []string
	for nickname := range game.GameData.PlayerPoints {
		nicknames = append(nicknames, nickname)
	}

	sort.Strings(nicknames)

	var result string
	for i, nickname := range nicknames {
		if i > 0 {
			result += ";"
		}
		result += fmt.Sprintf("%s:%d", nickname, game.GameData.PlayerPoints[nickname])
	}

	return result
}

func maskSentence(charactersSelected []string, sentenceToGuess string) string {
	masked := ""

	for _, char := range sentenceToGuess {
		if unicode.IsLetter(char) {
			lowerChar := unicode.ToLower(char)
			lowerSelected := make([]string, len(charactersSelected))
			for i, selectedChar := range charactersSelected {
				lowerSelected[i] = strings.ToLower(selectedChar)
			}

			if contains(lowerSelected, string(lowerChar)) {
				masked += string(char)
			} else {
				masked += "_"
			}
		} else {
			masked += string(char)
		}
	}

	return masked
}

func contains(slice []string, element string) bool {
	for _, el := range slice {
		if el == element {
			return true
		}
	}
	return false
}

func CreateSentenceGuessedMessage(game *structures.Game) string {
	magic := constants.MessageHeader
	messageType := constants.SentenceGuessed
	hint := game.GameData.Hint
	sentence := game.GameData.SentenceToGuess
	playerAndPoints := getPlayerNicknamesWithPoints(*game)
	messageBody := fmt.Sprintf("%s|%s|%s", hint, sentence, playerAndPoints)
	message := fmt.Sprintf("%s%03d%s%s\n", magic, len(messageBody), messageType, messageBody)
	return message
}

func getWinningPlayers(game *structures.Game) []string {
	var winningPlayers []string
	gameData := game.GameData

	for player, points := range gameData.PlayerPoints {
		if points >= constants.PointsNeededToWin {
			winningPlayers = append(winningPlayers, player)
		}
	}

	return winningPlayers
}

func CreateGameEndingMessage(game *structures.Game) string {
	magic := constants.MessageHeader
	messageType := constants.GameEnding
	winningPlayers := getWinningPlayers(game)
	messageBody := strings.Join(winningPlayers, ";")
	message := fmt.Sprintf("%s%03d%s%s\n", magic, len(messageBody), messageType, messageBody)
	return message
}

func CreatePingMessage() string {
	magic := constants.MessageHeader
	messageType := constants.Ping
	message := fmt.Sprintf("%s%03d%s\n", magic, 0, messageType)
	return message
}

func CreateCancelMessage() string {
	magic := constants.MessageHeader
	messageType := constants.Cancel
	message := fmt.Sprintf("%s%03d%s\n", magic, 0, messageType)
	return message
}

func CreateLobbyInfoMessage(gameStrings []string) string {
	message := strings.Join(gameStrings, ";")
	messageLength := fmt.Sprintf("%03d", len(message))
	finalMessage := constants.MessageHeader + messageLength + constants.LobbiesInfo + message + "\n"

	return finalMessage
}

func SendInfoAboutPendingUser(game structures.Game, invalidPlayer structures.Player) {
	message := createMessageAboutPendingUser(invalidPlayer)
	for _, player := range game.Players {
		if player != invalidPlayer {
			player.Socket.Write([]byte(message))
		}
	}
}

func createMessageAboutPendingUser(player structures.Player) string {
	magic := constants.MessageHeader
	messageType := constants.PendingUser
	messageLength := fmt.Sprintf("%03d", len(player.Nickname))
	finalMessage := magic + messageLength + messageType + player.Nickname + "\n"

	return finalMessage
}

func SendInfoAboutConnectedUser(game structures.Game, invalidPlayer structures.Player) {
	message := createMessageAboutConnectedUser(invalidPlayer)
	for _, player := range game.Players {
		if player != invalidPlayer {
			player.Socket.Write([]byte(message))
		}
	}
}

func createMessageAboutConnectedUser(player structures.Player) string {
	magic := constants.MessageHeader
	messageType := constants.ConnectedUser
	messageLength := fmt.Sprintf("%03d", len(player.Nickname))
	finalMessage := magic + messageLength + messageType + player.Nickname + "\n"

	return finalMessage
}

func CreateResendStateMessage(game *structures.Game, player structures.Player) string {
	magic := constants.MessageHeader
	messageType := constants.RetriveState
	players := getPlayerNicknamesWithPoints(*game)
	charactersSelectedSoFar := selectedCharactersFromGame(*game)
	hint := game.GameData.Hint
	status := 0 // did not play
	if game.GameData.PlayersPlayed[player.Nickname] {
		status = 1 // already played
	}
	maskedSentence := maskSentence(game.GameData.CharactersSelected, game.GameData.SentenceToGuess)
	messageBody := fmt.Sprintf("%s|%s|%s|%s|%d", players, charactersSelectedSoFar, maskedSentence, hint, status)
	finalMessage := fmt.Sprintf("%s%03d%s%s\n", magic, len(messageBody), messageType, messageBody)
	return finalMessage
}

func LobbyCannotBeStarted() string {
	magic := constants.MessageHeader
	messageType := constants.AlreadyInGame
	message := fmt.Sprintf("%s%03d%s\n", magic, 0, messageType)
	return message
}
