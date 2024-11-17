package main

import (
	"distributed-system/internal/system"
	"distributed-system/pkg/customerrors"
	"time"
)

func main() {
	sys := system.NewSystem(5)
	go sys.StartSystem()
	//time.Sleep(2 * time.Second)
	//sys.AddNodes(2)
	time.Sleep(18 * time.Second)
	if err := sys.CreateNewProcess([]float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7}); err != nil {
		customerrors.HandleError(err)
	}
	if err := sys.CreateNewProcess([]float64{0.8, 0.9}); err != nil {
		customerrors.HandleError(err)
	}

	select {}
}
