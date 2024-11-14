package main

import (
	"bufio"
	"distributed-system/pkg/utils"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	utils.LoadVEnv()
	protocol := os.Getenv("PROTOCOL")
	port := os.Getenv("PORT")
	conn, err := net.Dial(protocol, port)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Connected to server. Type messages and press Enter to send.")

	go func() {
		// Receiving messages from the server
		for {
			response, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				fmt.Println("Server disconnected:", err)
				os.Exit(0)
			}
			fmt.Print("Server response: " + response)
		}
	}()

	// Sending messages to the server
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.ToLower(text) == "exit" {
			fmt.Println("Exiting...")
			break
		}
		_, err := fmt.Fprintf(conn, text+"\n")
		if err != nil {
			fmt.Println("Error sending message:", err)
			break
		}
	}
}
