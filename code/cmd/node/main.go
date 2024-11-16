package main

import (
	"distributed-system/internal/node"
	"distributed-system/pkg/customerrors"
	"distributed-system/pkg/utils"
	"fmt"
	"os"
	"strconv"
)

func main() {
	args := os.Args
	utils.LoadVEnv()

	if len(args) <= 1 {
		fmt.Println("no nodeid provided")
		os.Exit(1)
	}

	protocol := os.Getenv("PROTOCOL")
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")
	address := fmt.Sprintf("%s:%s", host, port)
	id, _ := strconv.Atoi(args[1])
	fmt.Println("Id for new node is: ", id)
	//connect node
	nnode, err := node.SetUpConnection(protocol, address, id)
	if err != nil {
		customerrors.HandleError(err)
	}
	nnode.HandleNodeConnection()

}
