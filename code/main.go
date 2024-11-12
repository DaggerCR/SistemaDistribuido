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
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

// Estructura que representa un Nodo en la red distribuida
type Nodo struct {
	id       int
	carga    int   // Carga actual en el nodo
	procesos []int // Procesos asignados al nodo
}

// Sistema que mantiene el registro de nodos
type Sistema struct {
	nodos []*Nodo
	mutex sync.Mutex // Para sincronizar el acceso concurrente a los nodos
}

// Nueva instancia de Sistema con nodos iniciales
func nuevoSistema(numNodos int) *Sistema {
	sistema := &Sistema{}
	for i := 0; i < numNodos; i++ {
		sistema.nodos = append(sistema.nodos, &Nodo{id: i, carga: 0})
	}
	fmt.Printf("Sistema creado con %d nodos.\n", numNodos)
	return sistema
}

// Método para encontrar el nodo con menor carga
func (s *Sistema) nodoMenorCarga() *Nodo {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var nodoSeleccionado *Nodo
	for _, nodo := range s.nodos {
		if nodoSeleccionado == nil || nodo.carga < nodoSeleccionado.carga {
			nodoSeleccionado = nodo
		}
	}
	return nodoSeleccionado
}

// Método para asignar un proceso a un nodo
func (s *Sistema) asignarProceso(procesoID int) {
	nodo := s.nodoMenorCarga()
	s.mutex.Lock()
	nodo.procesos = append(nodo.procesos, procesoID)
	nodo.carga += 1 // Incrementa la carga al asignar un proceso
	s.mutex.Unlock()
	fmt.Printf("Proceso %d asignado al Nodo %d\n", procesoID, nodo.id)
}

// Función para mostrar el menú principal
func mostrarMenu() {
	fmt.Println("\n--- Menú Principal ---")
	fmt.Println("1. Crear sistema con nodos")
	fmt.Println("2. Asignar procesos a nodos")
	fmt.Println("3. Salir")
	fmt.Print("Seleccione una opción: ")
}

// Función para leer la entrada del usuario
func leerEntrada() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}

// Función principal para ejecutar el programa
func main() {
	rand.Seed(time.Now().UnixNano())

	var sistema *Sistema
	var sistemaCreado bool

	for {
		mostrarMenu()
		opcion := leerEntrada()

		switch opcion {
		case "1":
			// Crear sistema
			fmt.Print("Ingrese el número de nodos: ")
			numNodosStr := leerEntrada()
			numNodos, err := strconv.Atoi(numNodosStr)
			if err != nil || numNodos <= 0 {
				fmt.Println("Número de nodos inválido. Intente nuevamente.")
				continue
			}
			sistema = nuevoSistema(numNodos)
			sistemaCreado = true

		case "2":
			// Asignar procesos
			if !sistemaCreado {
				fmt.Println("Debe crear el sistema antes de asignar procesos.")
				continue
			}
			fmt.Print("Ingrese el número de procesos a asignar: ")
			numProcesosStr := leerEntrada()
			numProcesos, err := strconv.Atoi(numProcesosStr)
			if err != nil || numProcesos <= 0 {
				fmt.Println("Número de procesos inválido. Intente nuevamente.")
				continue
			}

			var wg sync.WaitGroup
			for i := 0; i < numProcesos; i++ {
				wg.Add(1)
				go func(procesoID int) {
					defer wg.Done()
					sistema.asignarProceso(procesoID)
					time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond) // Simula tiempo de ejecución
				}(i)
			}

			wg.Wait()

			// Mostrar estado final de cada nodo
			fmt.Println("\nEstado final de los nodos:")
			for _, nodo := range sistema.nodos {
				fmt.Printf("Nodo %d - Carga: %d, Procesos: %v\n", nodo.id, nodo.carga, nodo.procesos)
			}

		case "3":
			fmt.Println("Saliendo del programa.")
			return

		default:
			fmt.Println("Opción inválida. Intente nuevamente.")
		}
	}
}
