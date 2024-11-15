// Emulador de un sistema distribuido en Go
// Autor:
//		   - Daniel Sequeira
//		   - Josi Marín
//		   - Emanuel Rodriguez
// Fecha: 20 de noviembre

// Para ejecutar en windows:
// go build main.go
// .\main.exe

//Para ejecutar:
//...SistemaDistribuido\code\cmd\main> go run main.go

package main

import (
	"distributed-system/internal/system"
	"time"
)

func main() {
	sys := system.NewSystem()
	go sys.StartSystem()
	time.Sleep(10 * time.Second)
	go sys.CreateInitialNodes() //DELETE WHEN Crear y levantar nodos
	select {}
}
