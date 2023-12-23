package utils

import (
	"UPS_sem/constants"
	"fmt"
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

	message := fmt.Sprintf("%s%03d%s%s", magic, len(successStr), messageType, successStr)
	return message
}

func CanBeStarted(canBeStarted bool) string {
	magic := constants.MessageHeader
	messageType := constants.CanGameStart
	canBeStartedStr := "0"
	if canBeStarted {
		canBeStartedStr = "1"
	}

	message := fmt.Sprintf("%s%03d%s%s", magic, len(canBeStartedStr), messageType, canBeStartedStr)
	return message
}
