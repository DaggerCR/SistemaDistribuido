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
	for {
		fmt.Println("Menu:")
		fmt.Println("1. Start System")
		fmt.Println("2. View System log")
		fmt.Println("3. Exit")
		fmt.Print("Select an option: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			log.WithError(err).Error("Error reading user input")
			continue
		}
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			fmt.Print("Input the Max Node Load for the system: ")
			nodeLoadInput, err := reader.ReadString('\n')
			if err != nil {
				log.WithError(err).Error("Error reading Node Load input")
				continue
			}
			nodeLoadInput = strings.TrimSpace(nodeLoadInput)
			nodeLoad, err := strconv.Atoi(nodeLoadInput)

			if err != nil || nodeLoad <= 0 {
				log.Warn("Invalid input for Node Load")
				fmt.Println("Please use positive numbers")
				continue
			}

			sys := system.NewSystem(nodeLoad)
			go sys.StartSystem()
			log.WithField("Node Load", nodeLoad).Info("Starting system...")
			time.Sleep(18 * time.Second)

			if err := sys.CreateNewProcess([]float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7}); err != nil {
				customerrors.HandleError(err)
			}
			if err := sys.CreateNewProcess([]float64{0.8, 0.9}); err != nil {
				customerrors.HandleError(err)
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
			log.Info("Exit system by user decision")
			fmt.Println("Exit System...")
			return

		default:
			log.WithField("Option selected: ", input).Warn("Invalid option selected by user")
			fmt.Println("Invalid option")
		}
	}
}
