package system

import (
	"distributed-system/internal/node"
	"distributed-system/internal/procedure"
	"distributed-system/pkg/errors"
	"distributed-system/pkg/utils"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NodeId int
type Load int
type AccumulatedChecks int
type TimeStamp string

var clientProcesses []*exec.Cmd
var mu sync.Mutex

type System struct {
	procedures     []*procedure.Procedure
	loadBalance    map[NodeId]Load //maps node id to its load
	systemNodes    []node.Node
	healthRegistry map[NodeId]AccumulatedChecks //maps node id to the amount of heart-beat checks since last heart-beat from node
	nodesMu        sync.Mutex                   // used to lock systemNodes & healthRegistry
}

func NewSystem() *System {
	return &System{}
}

func (s *System) StartSystem() {
	fmt.Println("System is starting...")
	go s.OpenServer()
	for {
		time.Sleep(5 * time.Second)
		s.CheckHearbeat()
	}
}

func (s *System) OpenServer() {
	if err := utils.LoadVEnv(); err != nil {
		errors.HandleError(err)
	}
	protocol := os.Getenv("PROTOCOL")
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")
	address := fmt.Sprintf("%s:%s", host, port)
	listener, err := net.Listen(protocol, address)

	if err != nil {
		errors.HandleError(err)
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

func (s *System) HandleNodes(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		// Read data from client
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading from client:", err)
			return
		}
		// Process received data and send a response
		fmt.Println("Received data:", string(buf[:n]))
		conn.Write([]byte("OK-" + string(buf[:n])))
		s.HandleReceivedData(string(buf[:n]))
	}
}

func (s *System) HandleReceivedData(msj string) {
	parts := strings.Split(msj, "-")
	if len(parts) != 2 {
		fmt.Println("Invalid input format")
		return
	}
	id := parts[0]

	num, err := strconv.Atoi(id)
	if err != nil {
		// Handle the error if conversion fails
		fmt.Println("Error converting string to int:", err)
		return
	}
	nodeId := NodeId(num)
	command := parts[1]

	switch command {
	case "heartbeat":
		s.ReceiveHearbeat(nodeId)
	case "nodeUp":
		s.ReceiveHearbeat(nodeId)
	case "success":
		//
	case "failure":
		//
	}

}

func (s *System) ReceiveHearbeat(nodeId NodeId) {
	s.nodesMu.Lock() //Locked to avoid fatal error: concurrent map writes
	s.UpdateHealthRegistry(nodeId, 0)
	s.nodesMu.Unlock()
}

func (s *System) CheckHearbeat() {
	fmt.Println("Checking heartbeat of nodes...")
	//Locked to avoid reading while its being written in the health registry
	//Locked to avoid deleting node while another is being created, or fluctuations in systemNode size.
	s.nodesMu.Lock()

	for i := 0; i < len(s.systemNodes); i++ {
		s.healthRegistry[NodeId(s.systemNodes[i].Id())]++
	}
	fmt.Println(s.healthRegistry)
	for i := 0; i < len(s.systemNodes); i++ {
		if s.healthRegistry[NodeId(s.systemNodes[i].Id())] >= 3 {
			nodeId := NodeId(s.systemNodes[i].Id())
			fmt.Printf("Node %d will be killed.\n", nodeId)
			s.RemoveHealthRegistry(nodeId)
			s.RemoveSystemNode(i)
			fmt.Printf("New heartbeat registry is:")
			fmt.Println(s.healthRegistry)
		}
	}

	s.nodesMu.Unlock()
}

// EXAMPLE FUNCTION
func (s *System) CreateInitialNodes() {
	//In this example func, 5 nodes will be added, node 4 will be 'disconnected'
	for i := 0; i < 5; i++ {
		n, err := node.NewNode(i)
		if err != nil {
			fmt.Println("Error with node:", err)
		} else {
			//Locked to avoid multiple entries/updates at the same time.
			s.nodesMu.Lock()

			s.AppendNode(*n)
			s.UpdateHealthRegistry(NodeId(n.Id()), 0) //Initialize in 0

			s.nodesMu.Unlock()

			if n.Id() != 4 {
				go n.ConnectToSystem()
			}
		}
	}
}

// Procedures returns the slice of procedures in the system.
func (s *System) Procedures() []*procedure.Procedure {
	return s.procedures
}

// LoadBalance returns the load balance map in the system.
func (s *System) LoadBalance() map[NodeId]Load {
	return s.loadBalance
}

// SystemNodes returns the slice of nodes in the system.
func (s *System) SystemNodes() []node.Node {
	return s.systemNodes
}

// HealthRegistry returns the health registry map in the system.
func (s *System) HealthRegistry() map[NodeId]AccumulatedChecks {
	return s.healthRegistry
}

// Setters

// SetProcedures updates the procedures slice in the system.
func (s *System) SetProcedures(procedures []*procedure.Procedure) {
	s.procedures = procedures
}

// SetLoadBalance updates the load balance map in the system.
func (s *System) SetLoadBalance(loadBalance map[NodeId]Load) {
	s.loadBalance = loadBalance
}

// SetSystemNodes updates the system nodes slice in the system.
func (s *System) SetSystemNodes(systemNodes []node.Node) {
	s.systemNodes = systemNodes
}

// SetHealthRegistry updates the health registry map in the system.
func (s *System) SetHealthRegistry(healthRegistry map[NodeId]AccumulatedChecks) {
	s.healthRegistry = healthRegistry
}

// AppendProcedure adds a single procedure to the procedures slice.
func (s *System) AppendProcedure(proc *procedure.Procedure) {
	s.procedures = append(s.procedures, proc)
}

// AppendNode adds a single node to the systemNodes slice.
func (s *System) AppendNode(node node.Node) {
	s.systemNodes = append(s.systemNodes, node)
}

// Map Update Methods

// UpdateLoadBalance sets the load for a given NodeId in the loadBalance map.
func (s *System) UpdateLoadBalance(nodeID NodeId, load Load) {
	if s.loadBalance == nil {
		s.loadBalance = make(map[NodeId]Load)
	}
	s.loadBalance[nodeID] = load
}

// UpdateHealthRegistry updates the health registry for a given NodeId.
func (s *System) UpdateHealthRegistry(nodeID NodeId, check AccumulatedChecks) {
	if s.healthRegistry == nil {
		s.healthRegistry = make(map[NodeId]AccumulatedChecks)
	}
	s.healthRegistry[nodeID] = check
}

func (s *System) RemoveLoadBalance(nodeID NodeId) {
	delete(s.loadBalance, nodeID)
}

func (s *System) RemoveHealthRegistry(nodeID NodeId) {
	delete(s.healthRegistry, nodeID)
}

func (s *System) RemoveSystemNode(nodePosition int) {
	s.SetSystemNodes(append(s.systemNodes[:nodePosition], s.systemNodes[nodePosition+1:]...))
}
