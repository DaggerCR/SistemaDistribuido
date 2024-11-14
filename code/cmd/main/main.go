// Emulador de un sistema distribuido en Go
// Autor:
//		   - Daniel Sequeira
//		   - Josi Marín
//		   - Emanuel Rodriguez
// Fecha: 20 de noviembre

// Para ejecutar en windows:
// go build main.go
// .\main.exe

package main

import (
	"bufio"
	"distributed-system/pkg/errors"
	"distributed-system/pkg/utils"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {

	if err := utils.LoadVEnv(); err != nil {
		errors.HandleError(err)
	}
	protocol := os.Getenv("PROTOCOL")
	port := os.Getenv("PORT")
	//master := system.NewSystem()
	listener, err := net.Listen(protocol, port)

	if err != nil {
		errors.HandleError(err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("Cluster started on port %v\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Client connected:", conn.RemoteAddr())

	for {
		// Reading data from the client
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Client disconnected:", err)
			return
		}
		fmt.Printf("Received from client: %s", message)

		// Responding to the client
		response := strings.ToUpper(message)
		_, err = conn.Write([]byte(response))
		if err != nil {
			fmt.Println("Error sending response:", err)
			return
		}
	}
}
