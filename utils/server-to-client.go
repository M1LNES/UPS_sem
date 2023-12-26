package utils

import (
	"UPS_sem/constants"
	"UPS_sem/structures"
	"fmt"
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
	var nicknames string

	for _, player := range game.Players {
		if nicknames != "" {
			nicknames += ";"
		}
		nicknames += fmt.Sprintf("%s:%d", player.Nickname, game.GameData.PlayerPoints[player])
	}

	return nicknames
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

func getWinningPlayers(game *structures.Game) []structures.Player {
	var winningPlayers []structures.Player
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
	var winningNicknames []string
	for _, player := range winningPlayers {
		winningNicknames = append(winningNicknames, player.Nickname)
	}
	messageBody := strings.Join(winningNicknames, ";")
	message := fmt.Sprintf("%s%03d%s%s\n", magic, len(messageBody), messageType, messageBody)
	return message
}
