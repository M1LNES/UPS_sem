package main

import (
	"UPS_sem/constants"
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	connHost = "0.0.0.0"
	connPort = "10000"
	connType = "tcp"
)

var clientsMap = make(map[net.Conn]string)

func main() {
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

func handleConnection(client net.Conn) {
	defer client.Close()

	reader := bufio.NewReader(client)

	for {
		readBuffer, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Println("Zabijim: ", clientsMap[client])
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
		client.Write(readBuffer)
	}
}

func handleMessage(message string, client net.Conn) {
	messageType := message[len(constants.MessageHeader)+constants.MessageLengthFormat : len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength]
	if messageType == "nick" {
		clientsMap[client] = message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:]
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

	// Print or use the length, type, and the rest of the message as needed
	fmt.Printf("Magic: %s\n", magic)
	fmt.Printf("Length: %d\n", length)
	fmt.Printf("Type: %s\n", messageType)
	fmt.Printf("Message: %s\n", message[len(constants.MessageHeader)+constants.MessageLengthFormat+constants.MessageTypeLength:])

	return true
}
