package main

import (
	"bufio"
	"distributed-system/internal/system"
	"distributed-system/pkg/customerrors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	logFile, err := os.OpenFile("system.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

	reader := bufio.NewReader(os.Stdin)
	var sys system.System
	systemUp := false
	for {
		time.Sleep(500 * time.Millisecond)
		fmt.Println("Menu:")
		fmt.Println("1. Start System")
		fmt.Println("2. View System log")
		fmt.Println("3. Create task simulated")
		fmt.Println("4. Add nodes")
		fmt.Println("5. Exit")
		fmt.Print("Select an option: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			log.WithError(err).Error("Error reading user input")
			continue
		}
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			if !systemUp {
				sys = *system.NewSystem()
				go sys.StartSystem()
				log.WithField("System start", true).Info("Starting system...")
				systemUp = true
			} else {
				fmt.Println("\nSystem already up ")
			}
		case "2":
			logFileContent, err := os.ReadFile("system.log")
			if err != nil {
				log.WithError(err).Error("Error reading log file")
				fmt.Println("Error reading log file", err)
				continue
			}
			fmt.Println("\n====== LOG =====")
			fmt.Println(string(logFileContent))
			fmt.Println("===============")
		case "3":
			if systemUp {
				if err := sys.CreateNewProcess([]float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7}); err != nil {
					customerrors.HandleError(err)
					continue
				}
				if err := sys.CreateNewProcess([]float64{0.8, 0.9}); err != nil {
					customerrors.HandleError(err)
					continue
				}
				if err := sys.CreateNewProcess(
					[]float64{
						0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0,
						1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 2.0,
						2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 3.0,
						3.1, 3.2, 3.3, 3.4, 3.5,
					}); err != nil {
					customerrors.HandleError(err)
					continue
				}
				if err := sys.CreateNewProcess(
					[]float64{
						0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0,
						1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 2.0,
						2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 3.0,
						3.1, 3.2, 3.3, 3.4, 3.5,
					}); err != nil {
					customerrors.HandleError(err)
					continue
				}
				if err := sys.CreateNewProcess([]float64{
					3.5, 3.6, 3.7, 3.8, 3.9, 4.0, 4.1, 4.2, 4.3, 4.4,
					4.5, 4.6, 4.7,
				}); err != nil {
					customerrors.HandleError(err)
					continue
				}

			} else {
				fmt.Println("\nNo system created yet")
			}
		case "4":
			if systemUp {
				fmt.Println("Input number of nodes to create: \t")
				input, err := reader.ReadString('\n')
				if err != nil {
					log.WithError(err).Error("Error reading user input")
					continue
				}
				input = strings.TrimSpace(input)
				numNodes, err := strconv.Atoi(input)
				if err != nil {
					log.WithError(err).Error("Error reading user input")
					continue
				}
				log.WithField("Created nodes with quantity ", numNodes).Info("Starting nodes...")
				sys.AddNodes(numNodes)
				continue
			} else {
				fmt.Println("\nNo system created yet")
			}
		case "5":
			log.Info("Exit system by user decision")
			fmt.Println("\nExit System...")
			return

		default:
			log.WithField("Option selected: ", input).Warn("Invalid option selected by user")
			fmt.Println("Invalid option")
		}
	}
}
