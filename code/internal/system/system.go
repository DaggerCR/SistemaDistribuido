package system

import (
	"bytes"
	"distributed-system/internal/message"
	tscheduler "distributed-system/internal/task_scheduler"
	"distributed-system/pkg/customerrors"
	"distributed-system/pkg/utils"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type System struct {
	systemNodes    map[utils.NodeId]net.Conn                // Regular map for system nodes
	healthRegistry map[utils.NodeId]utils.AccumulatedChecks // Regular map for node health
	taskScheduler  tscheduler.TScheduler
	mu             sync.RWMutex // Protects loadBalance, systemNodes, and healthRegistry
	muConn         sync.RWMutex // Protects systemNodes
}

// NewSystem initializes the system.
func NewSystem(defaultMaxNodeLoad int) *System {
	return &System{
		systemNodes:    make(map[utils.NodeId]net.Conn),
		healthRegistry: make(map[utils.NodeId]utils.AccumulatedChecks),
		taskScheduler:  *tscheduler.NewTScheduler(defaultMaxNodeLoad),
	}
}

// StartSystem starts the system and its heartbeat checker.
func (s *System) StartSystem() {
	fmt.Println("System is starting...")
	go s.OpenServer()
	go s.StartHeartbeatChecker()

	select {} // Block forever
}

// OpenServer starts listening for incoming connections.
func (s *System) OpenServer() {
	if err := utils.LoadVEnv(); err != nil {
		customerrors.HandleError(err)
	}
	protocol := os.Getenv("PROTOCOL")
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")

	address := fmt.Sprintf("%s:%s", host, port)
	fmt.Printf("Coneccion desde: %v,%v\n", protocol, address)
	listener, err := net.Listen(protocol, address)

	if err != nil {
		customerrors.HandleError(err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("System started on port %v\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go s.HandleNodes(conn)
	}
}

func (s *System) AddNodes(quantity int) {
	for i := 0; i < quantity; i++ {
		// Dynamically construct the path to main.go
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting working directory:", err)
			return
		}
		cmdPath := filepath.Join(wd, "cmd", "node", "main.go")

		// Create the command
		cmd := exec.Command("go", "run", cmdPath, fmt.Sprint(s.GetRandomIdNotInNodes()))

		// Capture output for debugging
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		// Start the command
		if err := cmd.Start(); err != nil {
			fmt.Println("Error starting command:", err)
			fmt.Println(stderr.String())
			return
		}

		fmt.Printf("Command is running with PID: %d\n", cmd.Process.Pid)

		// Use a goroutine to handle command completion
		go func(c *exec.Cmd) {
			err := c.Wait()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					fmt.Printf("Command exited with code: %d\n", exitErr.ExitCode())
				} else {
					fmt.Println("Error waiting for command:", err)
				}
				fmt.Println("Command stderr:", stderr.String())
			} else {
				fmt.Println("Command finished successfully")
				fmt.Println("Command stdout:", out.String())
			}
		}(cmd)
	}
}

func (s *System) GetRandomIdNotInNodes() int {
	for {
		// Generate a random number using GenRandomUint
		num := utils.GenRandomUint()
		// Check if the number is not in the map
		if _, exists := s.GetConnection(utils.NodeId(num)); !exists {
			return int(num)
		}
	}
}

// HandleNodes processes data from connected nodes.
func (s *System) HandleNodes(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading from client:", err)
			return
		}
		s.HandleReceivedData(buf[:n], n, conn)
	}
}

// HandleReceivedData processes messages received from nodes.
func (s *System) HandleReceivedData(buffer []byte, size int, conn net.Conn) {
	msg, err := message.InterpretMessage(buffer, size)
	if err != nil {
		fmt.Println("Erroneous message received, ignoring...")
		return
	}
	if msg.Action != message.Heartbeat {
		fmt.Printf("Master received message from: %v, content: %v\n", msg.Sender, msg.Content)
	}
	switch msg.Action {
	case message.Heartbeat:
		s.ReceiveHeartbeat(utils.NodeId(msg.Sender))
	case message.NotifyNodeUp:
		s.ReceiveHeartbeat(utils.NodeId(msg.Sender))
		s.ReceiveNodeUp(utils.NodeId(msg.Sender), conn)
	default:
		fmt.Println("Unknown action received.")
	}
}

// ReceiveHeartbeat resets the heartbeat count for a node.
func (s *System) ReceiveHeartbeat(nodeId utils.NodeId) {
	s.UpdateHealthRegistry(nodeId, 0)
}

// ReceiveNodeUp handlefs a node coming online.
func (s *System) ReceiveNodeUp(nodeId utils.NodeId, conn net.Conn) {
	fmt.Printf("Node %d is online.\n", nodeId)
	s.AppendNode(nodeId, conn)
	//s.AddToLoadBalance(nodeId)
}

// StartHeartbeatChecker periodically checks node health.
func (s *System) StartHeartbeatChecker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.CheckHeartbeat()
	}
}

func (s *System) CheckHeartbeat() {
	fmt.Println("Checking heartbeat of nodes...")

	s.mu.Lock()     // Lock for healthRegistry
	s.muConn.Lock() // Lock for systemNodes
	defer s.mu.Unlock()
	defer s.muConn.Unlock()

	for nodeId, checks := range s.healthRegistry {
		newChecks := checks + 1
		s.healthRegistry[nodeId] = newChecks

		if newChecks >= 3 {
			fmt.Printf("Node %d will be killed.\n", nodeId)
			delete(s.healthRegistry, nodeId)
			delete(s.systemNodes, nodeId)
		}
	}
}

/*
func (s *System) CreateNewProcess(array []float64) {
	newProcessId := s.GetRandomIdNotInProcesses()
	s.processMu.Lock()
	defer s.processMu.Unlock()
}
*/

// UpdateHealthRegistry updates the health registry for a node.
func (s *System) UpdateHealthRegistry(nodeId utils.NodeId, checks utils.AccumulatedChecks) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthRegistry[nodeId] = checks
}

// AppendNode safely adds a node and its connection to the systemNodes map.
func (s *System) AppendNode(nodeId utils.NodeId, conn net.Conn) {
	s.muConn.Lock()
	defer s.muConn.Unlock()
	s.systemNodes[nodeId] = conn
}

// RemoveSystemNode safely removes a node from the systemNodes map.
func (s *System) RemoveSystemNode(nodeId utils.NodeId) {
	s.muConn.Lock()
	defer s.muConn.Unlock()
	delete(s.systemNodes, nodeId)
}

// GetConnection safely retrieves a connection by utils.NodeId.
func (s *System) GetConnection(nodeId utils.NodeId) (net.Conn, bool) {
	s.muConn.RLock()
	defer s.muConn.RUnlock()
	conn, exists := s.systemNodes[nodeId]
	return conn, exists
}
