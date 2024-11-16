package main

import (
	"distributed-system/internal/system"
	"time"
)

func main() {
	sys := system.NewSystem()
	go sys.StartSystem()
	time.Sleep(4 * time.Second)
	sys.AddNodes(2)

	//go sys.CreateInitialNodes() //DELETE WHEN Crear y levantar nodos
	select {}
}
