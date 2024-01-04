package utils

import (
	"UPS_sem/constants"
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
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

func ValidateConfig() bool {
	if constants.MaxPlayers <= 2 {
		fmt.Println("MaxPlayers should be greater than 2")
		return false
	}

	// Validate RoomsCount
	if constants.RoomsCount <= 1 {
		fmt.Println("RoomsCount should be greater than 1")
		return false
	}

	// Validate ConnType
	if constants.ConnType != "tcp" {
		fmt.Println("ConnType should be 'tcp'")
		return false
	}

	// Validate ConnHost
	if constants.ConnHost != "localhost" {
		if net.ParseIP(constants.ConnHost) == nil {
			fmt.Println("ConnHost should be an IP address or 'localhost'")
			return false
		}
	}

	port, err := strconv.Atoi(constants.ConnPort)
	if err != nil {
		fmt.Println("ConnPort should be a valid port number")
		return false
	}

	if port < 1 {
		fmt.Println("ConnPort should be natural number")
		return false
	}

	dictionaryFilePath := "./dictionary/" + constants.DictionaryFile
	_, err = os.Stat(dictionaryFilePath)
	if err != nil {
		fmt.Printf("Dictionary file '%s' does not exist\n", constants.DictionaryFile)
		return false
	}

	if !validateDictionaryFormat(dictionaryFilePath) {
		return false
	}

	return true
}

func validateDictionaryFormat(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening dictionary file: %v\n", err)
		return false

	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !validDictionaryEntry.MatchString(line) {
			fmt.Printf("Invalid dictionary entry format: %s\n", line)
			return false
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading dictionary file: %v\n", err)
		return false
	}

	return true
}

var validDictionaryEntry = regexp.MustCompile(`^[a-zA-Z0-9:.,'"\s-]+;[a-zA-Z0-9:.,'"\s-]+$`)

func IsCharacterValid(letter string) bool {
	if len(letter) != 1 {
		return false
	}

	return letter >= "a" && letter <= "z"
}
