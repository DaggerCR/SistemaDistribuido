package main

import (
	"bufio"
	"distributed-system/internal/system"
	"distributed-system/logs"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	logs.Initialize()
	reader := bufio.NewReader(os.Stdin)
	var sys system.System
	systemUp := false
	for {
		logs.Log.WithField("Starting Program with System up", systemUp).Info("Running distributive system...")
		time.Sleep(500 * time.Millisecond)
		fmt.Print("\n\n")
		fmt.Println("╔════════════════════════════════╗")
		fmt.Println("║       Distributed System       ║")
		fmt.Println("║                                ║")
		fmt.Println("║Menu                            ║")
		fmt.Println("║ 1. Start System                ║")
		fmt.Println("║ 2. View System log             ║")
		fmt.Println("║ 3. Create task simulated       ║")
		fmt.Println("║ 4. Add nodes                   ║")
		fmt.Println("║ 5. Exit                        ║")
		fmt.Println("╚════════════════════════════════╝")
		fmt.Print("Select an option: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			logs.Log.WithField("Error reading menu option", err).Error("Invalid input")
			continue
		}
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			if !systemUp {
				sys = *system.NewSystem()
				go sys.StartSystem()
				logs.Log.Info("===========================================================================================================Starting system===========================================================================================================")
				systemUp = true
			} else {
				logs.Log.WithField("The system couldn't be initialized", err).Warn("System already up")
				fmt.Println("\nSystem already up")
				continue
			}
		case "2":
			logFileContent, err := os.ReadFile("../../logs/system.log")
			if err != nil {
				logs.Log.WithField("Error reading log file", err).Error("log file error")
				continue
			}
			fmt.Println("\n====== LOG =====")
			fmt.Println(string(logFileContent))
			fmt.Println("===============")
		case "3":
			if systemUp {
				sys.CreateNewProcess([]float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7})
				sys.CreateNewProcess([]float64{0.8, 0.9, 0, 10})
				sys.CreateNewProcess([]float64{
					3.5, 3.6, 3.7, 3.8, 3.9, 4.0, 4.1, 4.2, 4.3, 4.4,
					4.5, 4.6, 4.7,
				})
				sys.CreateNewProcess([]float64{0.5, 0.5, 1.0, 1.0, 2.0})
				sys.CreateNewProcess([]float64{0.25, 0.25, 0.25})
				continue

			} else {
				fmt.Println("\nNo system created yet")
				logs.Log.WithField("No system created yet", err).Warn("couldnt create a task, the system wasn't up")
			}
		case "4":
			if systemUp {
				fmt.Println("Input number of nodes to create: \t")
				input, err := reader.ReadString('\n')
				if err != nil {
					logs.Log.WithField("Error reading number of nodes: ", err).Error("invalid input")
					continue
				}
				input = strings.TrimSpace(input)
				numNodes, err := strconv.Atoi(input)
				if err != nil {
					logs.Log.WithField("Error reading number of nodes: ", err).Error("invalid input")
					continue
				}
				logs.Log.WithField("Created nodes with quantity ", numNodes).Info("Starting nodes..")
				sys.AddNodes(numNodes)
				continue
			} else {
				fmt.Println("\nNo system created yet")
				logs.Log.WithField("No system created yet", systemUp).Warn("no system found")
			}
		case "5":
			logs.Log.Info("Exit program by user decision")
			fmt.Println("\nExit program...")
			return

		default:
			logs.Log.WithField("Option selected: ", input).Warn("Invalid option selected by user")
			fmt.Println("Invalid option")
		}
	}
}
