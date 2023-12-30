package utils

import (
	"UPS_sem/constants"
	"strconv"
)

func IsLengthValid(message string) bool {
	if !hasMinimumLength(message) {
		return false
	}

	if !hasValidMagic(message) {
		return false
	}

	if !hasValidMessageLength(message) {
		return false
	}

	return true
}

func hasMinimumLength(message string) bool {
	minLength := len(constants.MessageHeader) + constants.MessageTypeLength + constants.MessageLengthFormat
	return len(message) >= minLength
}

func hasValidMagic(message string) bool {
	magic := message[:len(constants.MessageHeader)]
	if magic != constants.MessageHeader {
		return false
	}
	return true
}

func hasValidMessageLength(message string) bool {
	lengthStr := message[len(constants.MessageHeader) : len(constants.MessageHeader)+constants.MessageLengthFormat]
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return false
	}

	expectedLength := len(message) - len(constants.MessageHeader) - constants.MessageLengthFormat - constants.MessageTypeLength
	if length != expectedLength {
		return false
	}

	return true
}
