package menu

import (
	"distributed-system/pkg/utils"
	"fmt"
)

func displayMenuOptions() {
	fmt.Println("\n--- Menú Principal ---")
	fmt.Println("1. Crear sistema con nodos")
	fmt.Println("2. Asignar procesos a nodos")
	fmt.Println("3. Simular fallo de nodo")
	fmt.Println("4. Salir")
	fmt.Print("Seleccione una opción: ")
}

// Función principal para ejecutar el programa
func StartMenu() {
	displayMenuOptions()
	//var existsMaster bool
	//var master system.System
	for {
		displayMenuOptions()
		opcion := utils.ReadEntry()
		switch opcion {

		case "1":
			for i := 0; i < 10; i++ {
				fmt.Printf("random number: %v", utils.GenRandomUint())
			}
		case "2":
		case "3":
		case "4":
			fmt.Println("Saliendo del programa.")
			return
		default:
			fmt.Println("Opción inválida. Intente nuevamente.")
		}
	}
}
