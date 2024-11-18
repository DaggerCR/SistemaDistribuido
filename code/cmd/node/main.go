package main

import (
	"bufio"
	"distributed-system/internal/node"
	"distributed-system/pkg/customerrors"
	"distributed-system/pkg/utils"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func main() {

	log := logrus.New()
	logFile, err := os.OpenFile("node.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFile.Close()
	log.SetOutput(logFile) // write on log file
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(logrus.InfoLevel)

	args := os.Args
	utils.LoadVEnv()

	var id int
	if len(args) <= 1 {
		fmt.Println("No node ID provided on the argument. Please enter one:")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			log.WithError(err).Error("Error reading user input")
			os.Exit(1)
		}

		input = strings.TrimSpace(input)
		id, err = strconv.Atoi(input)
		if err != nil {
			fmt.Println("Invalid node ID. Please try again and enter a valid number.")
			log.WithError(err).Error("Invalid node ID.")
			os.Exit(1)
		}
	} else {
		var err error
		id, err = strconv.Atoi(args[1])
		if err != nil {
			fmt.Println("Invalid node ID provided as argument.")
			log.WithError(err).Error("Invalid node ID provided as argument.")
			os.Exit(1)
		}
	}

	protocol := os.Getenv("PROTOCOL")
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")
	address := fmt.Sprintf("%s:%s", host, port)

	fmt.Println("Node ID:", id)
	fmt.Println("Address:", address)
	fmt.Println("Protocol:", protocol)
	fmt.Println("Id for the new node is: ", id)

	log.WithField("Node ID", id).Info("ID selected by user")
	log.WithField("Adress", id).Info("Adress provided by user")
	log.WithField("Protocol", id).Info("ID selected by user")
	//connect node
	nnode, err := node.SetUpConnection(protocol, address, id)
	if err != nil {
		customerrors.HandleError(err)
	}
	nnode.HandleNodeConnection()

}
