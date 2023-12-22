package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	serverHost = "localhost"
	serverPort = "10000"
	serverType = "tcp"
)

func main() {
	fmt.Println("Connecting to " + serverType + " server " + serverHost + ":" + serverPort)

	server, err := net.Dial(serverType, serverHost+":"+serverPort)
	if err != nil {
		fmt.Println("Error connecting:", err.Error())
		os.Exit(1)
	}
	defer server.Close()

	go handleServerResponse(server)

	reader := bufio.NewReader(os.Stdin)

	for {
		input, _ := reader.ReadString('\n')

		// Trim the newline character from the input
		input = strings.TrimRight(input, "\n")

		server.Write([]byte(input + "\n"))
	}
}

func handleServerResponse(server net.Conn) {
	for {
		message, err := bufio.NewReader(server).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from server:", err.Error())
			break
		}

		fmt.Print(message)
	}
}
