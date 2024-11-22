package main

import (
	"bufio"
	"distributed-system/internal/node"
	"distributed-system/logs"
	"distributed-system/pkg/customerrors"
	"distributed-system/pkg/utils"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {

	args := os.Args
	utils.LoadVEnv()
	logs.Initialize()

	var id int
	if len(args) <= 1 {
		fmt.Println("No node ID provided on the argument. Please enter one:")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			logs.Log.WithError(err).Error("Error reading user input")
			os.Exit(1)
		}

		input = strings.TrimSpace(input)
		id, err = strconv.Atoi(input)
		if err != nil {
			logs.Log.WithError(err).Error("Invalid node ID.")
			os.Exit(1)
		}
	} else {
		var err error
		id, err = strconv.Atoi(args[1])
		if err != nil {
			logs.Log.WithError(err).Error("Invalid node ID provided as argument.")
			os.Exit(1)
		}
	}

	protocol := os.Getenv("PROTOCOL")
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")
	address := fmt.Sprintf("%s:%s", host, port)

	logs.Log.WithField("Node ID", id).Info("ID selected by user")
	logs.Log.WithField("Adress", id).Info("Adress provided by user")
	logs.Log.WithField("Protocol", id).Info("Protocol selected by user")

	//connect node
	nnode, err := node.SetUpConnection(protocol, address, id)
	if err != nil {
		customerrors.HandleError(err)
		logs.Log.WithError(err).Error("Error setting up connection with the node")
	}
	nnode.HandleNodeConnection()

}
