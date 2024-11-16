package system

import (
	"bytes"
	"distributed-system/internal/message"
	"distributed-system/internal/process"
	"distributed-system/pkg/customerrors"
	"distributed-system/pkg/utils"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type NodeId int
type Load int
type AccumulatedChecks int

type System struct {
	processes      []*process.Process
	loadBalance    map[NodeId]Load              // Regular map for load balancing
	systemNodes    map[NodeId]net.Conn          // Regular map for system nodes
	healthRegistry map[NodeId]AccumulatedChecks // Regular map for node health

	mu        sync.RWMutex // Protects loadBalance, systemNodes, and healthRegistry
	processMu sync.Mutex   // Protects processes slice
}

// NewSystem initializes the system.
func NewSystem() *System {
	return &System{
		loadBalance:    make(map[NodeId]Load),
		systemNodes:    make(map[NodeId]net.Conn),
		healthRegistry: make(map[NodeId]AccumulatedChecks),
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
		cmd := exec.Command("go", "run", cmdPath, fmt.Sprint(GetRandomIdNotInNodes(s.systemNodes)))

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

func GetRandomIdNotInNodes(m map[NodeId]net.Conn) int {
	for {
		// Generate a random number using GenRandomUint
		num := utils.GenRandomUint()
		// Check if the number is not in the map
		if _, exists := m[NodeId(num)]; !exists {
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
		s.HandleReceivedData(buf[:n], n)
	}
}

// HandleReceivedData processes messages received from nodes.
func (s *System) HandleReceivedData(buffer []byte, size int) {
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
		s.ReceiveHeartbeat(NodeId(msg.Sender))
	case message.NotifyNodeUp:
		s.ReceiveHeartbeat(NodeId(msg.Sender))
		s.ReceiveNodeUp(NodeId(msg.Sender))
	default:
		fmt.Println("Unknown action received.")
	}
}

// ReceiveHeartbeat resets the heartbeat count for a node.
func (s *System) ReceiveHeartbeat(nodeId NodeId) {
	s.UpdateHealthRegistry(nodeId, 0)
}

// ReceiveNodeUp handlefs a node coming online.
func (s *System) ReceiveNodeUp(nodeId NodeId) {
	fmt.Printf("Node %d is online.\n", nodeId)
}

// StartHeartbeatChecker periodically checks node health.
func (s *System) StartHeartbeatChecker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.CheckHeartbeat()
	}
}

// CheckHeartbeat increments health counters and removes unresponsive nodes.
func (s *System) CheckHeartbeat() {
	fmt.Println("Checking heartbeat of nodes...")

	s.mu.Lock()
	defer s.mu.Unlock()

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

// Processes returns the slice of processes in the system.
func (s *System) Processes() []*process.Process {
	s.processMu.Lock()
	defer s.processMu.Unlock()
	return s.processes
}

// SetProcesses updates the processes slice in the system.
func (s *System) SetProcesses(processes []*process.Process) {
	s.processMu.Lock()
	defer s.processMu.Unlock()
	s.processes = processes
}

// UpdateLoadBalance sets the load for a given NodeId.
func (s *System) UpdateLoadBalance(nodeId NodeId, load Load) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadBalance[nodeId] = load
}

// GetLoadBalance retrieves the load for a given NodeId.
func (s *System) GetLoadBalance(nodeId NodeId) (Load, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	load, ok := s.loadBalance[nodeId]
	return load, ok
}

// RemoveLoadBalance removes a node from the load balance map.
func (s *System) RemoveLoadBalance(nodeId NodeId) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.loadBalance, nodeId)
}

// UpdateHealthRegistry updates the health registry for a node.
func (s *System) UpdateHealthRegistry(nodeId NodeId, checks AccumulatedChecks) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthRegistry[nodeId] = checks
}

// AppendNode adds a node to the system.
func (s *System) AppendNode(nodeId NodeId, conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.systemNodes[nodeId] = conn
}

// RemoveSystemNode removes a node from the system.
func (s *System) RemoveSystemNode(nodeId NodeId) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.systemNodes, nodeId)
}

// ProbeLessLoadedNodes retrieves nodes with the least load.
func (s *System) ProbeLessLoadedNodes(amount int) []NodeId {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect all nodes and their loads
	type nodeLoad struct {
		nodeId NodeId
		load   Load
	}
	var nodes []nodeLoad
	for nodeId, load := range s.loadBalance {
		nodes = append(nodes, nodeLoad{nodeId, load})
	}

	// Sort nodes by load in ascending order
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].load < nodes[j].load
	})

	// Return the `amount` least loaded nodes
	result := make([]NodeId, 0, amount)
	for i := 0; i < len(nodes) && i < amount; i++ {
		result = append(result, nodes[i].nodeId)
	}
	return result
}
