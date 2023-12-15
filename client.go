package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
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

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Text to send: ")
		input, _ := reader.ReadString('\n')

		server.Write([]byte(input))

		message, err := bufio.NewReader(server).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from server:", err.Error())
			break
		}

		log.Print("Server reply: ", message)
	}
}
