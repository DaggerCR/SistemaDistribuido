package system

import (
	"bytes"
	"distributed-system/internal/message"
	"distributed-system/internal/task"
	tscheduler "distributed-system/internal/tscheduler"
	"distributed-system/logs"
	"distributed-system/pkg/customerrors"
	"distributed-system/pkg/utils"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type System struct {
	systemNodes    map[utils.NodeId]net.Conn                // Regular map for system nodes
	healthRegistry map[utils.NodeId]utils.AccumulatedChecks // Regular map for node health
	taskScheduler  tscheduler.TScheduler
	mu             sync.RWMutex // Protects loadBalance, systemNodes, and healthRegistry
	muConn         sync.RWMutex // Protects systemNodes
}

// NewSystem initializes the system.
func NewSystem() *System {
	return &System{
		systemNodes:    make(map[utils.NodeId]net.Conn),
		healthRegistry: make(map[utils.NodeId]utils.AccumulatedChecks),
	}
}

// StartSystem starts the system and its heartbeat checker.
func (s *System) StartSystem() {
	logs.Initialize()
	logs.Log.Info("[INFO] System is starting...")
	go s.OpenServer()
	go s.StartHeartbeatChecker()
	go s.WaitlistRetry()

	select {} // Block forever
}

// OpenServer starts listening for incoming connections.
func (s *System) OpenServer() {
	logs.Initialize()
	if err := utils.LoadVEnv(); err != nil {
		customerrors.HandleError(err)
		os.Exit(1)
	}
	s.taskScheduler = *tscheduler.NewTScheduler()
	protocol := os.Getenv("PROTOCOL")
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")

	address := fmt.Sprintf("%s:%s", host, port)
	listener, err := net.Listen(protocol, address)

	if err != nil {
		customerrors.HandleError(err)
		os.Exit(1)
	}
	defer listener.Close()

	logs.Log.WithField("Cluster started at port", address).Info("[INFO]")
	for {
		conn, err := listener.Accept()
		if err != nil {
			logs.Log.WithField("accepting connection", err).Error("[ERROR]")
			continue
		}
		go s.HandleNodes(conn)
	}
}

func (s *System) AddNodes(quantity int) {
	logs.Initialize()
	for i := 0; i < quantity; i++ {
		// Dynamically construct the path to node/main.go
		wd, err := os.Getwd()
		if err != nil {
			logs.Log.WithField("Error getting working directory", err).Error("[ERROR]")
			return
		}
		cmdPath := filepath.Join(wd, "cmd", "node", "main.go")
		/*
			parentDir := filepath.Dir(wd)
			cmdPath = filepath.Join(parentDir, "node", "main.go")
		*/
		// Create the command
		cmd := exec.Command("go", "run", cmdPath, fmt.Sprint(s.GetRandomIdNotInNodes()))
		// Capture output for debugging
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		// Start the command
		if err := cmd.Start(); err != nil {
			logs.Log.WithField("Error starting command", err).Error("[ERROR]", stderr.String())
			return
		}
		logs.Log.WithField("Command is running with PID", cmd.Process.Pid).Info("[INFO]")

		// Use a goroutine to handle command completion
		go func(c *exec.Cmd) {
			err := c.Wait()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					logs.Log.WithField("Command exited with code", exitErr.ExitCode()).Error("[Error]")

				} else {
					logs.Log.WithField("Error waiting for command", err).Error("[ERROR]")
				}
				logs.Log.WithField("Command stderr", stderr.String()).Info("[INFO]")
			} else {
				logs.Log.Info("[INFO] Command finished successfully")
				logs.Log.WithField("Command stdout:", out.String()).Info("[INFO]")
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
	logs.Initialize()
	defer conn.Close()
	for {
		buffer, err := message.RecieveMessage(conn)
		if err != nil {
			//logs.Log.WithField("Error reading from client", err).Error("[ERROR]")
			return
		}
		go s.HandleReceivedData(buffer, conn)
	}
}

// HandleReceivedData processes messages received from nodes.
func (s *System) HandleReceivedData(buffer []byte, conn net.Conn) {
	logs.Initialize()
	msg, err := message.InterpretMessage(buffer)
	if err != nil {
		logs.Log.WithField("Erroneous message received, ignoring...", err).Error("[ERROR]")
		return
	}
	switch msg.Action {
	case message.Heartbeat:
		s.ReceiveHeartbeat(utils.NodeId(msg.Sender))
	case message.NotifyNodeUp:
		logs.Log.Info("[INFO] Notify Node Up")
		s.ReceiveHeartbeat(utils.NodeId(msg.Sender))
		s.ReceiveNodeUp(utils.NodeId(msg.Sender), conn)
	case message.ActionSuccess:
		logs.Log.Info("[INFO] Action Success")
		if msg.Task.Id != -1 { //-1 as task id to tell that message didnt have task, this because go converts missing values in any declaration to "default" values (this case tasks )
			s.ReceiveSuccess(utils.NodeId(msg.Sender), buffer, msg.Task)
		} else {
			s.ReceiveSuccess(utils.NodeId(msg.Sender), buffer)
		}
	case message.ReturnedRes:
		{
			logs.Log.Info("[INFO] ReturnedRes")
			//content only contains the float64 result parsed as a string
			s.ReceiveReturnedRes(msg.Task, utils.NodeId(msg.Sender), msg.Content, conn)
		}
	default:
		logs.Log.WithField("Unknown action received.", msg.Content).Warn("[WARN]")

	}
}

// ReceiveHeartbeat resets the heartbeat count for a node.
func (s *System) ReceiveHeartbeat(nodeId utils.NodeId) {
	s.UpdateHealthRegistry(nodeId, 0)
}

// ReceiveNodeUp handles a node coming online.
func (s *System) ReceiveNodeUp(nodeId utils.NodeId, conn net.Conn) {
	logs.Initialize()
	logs.Log.WithField("the Node is online, id", nodeId).Info("[INFO]")
	s.AppendNode(nodeId, conn)
	s.muConn.Lock()
	logs.Log.WithField("Quantity of nodes now is: ", len(s.systemNodes)).Info("[INFO]")
	s.muConn.Unlock()
	s.taskScheduler.AddToLoadBalance(nodeId)
}

func (s *System) ReceiveSuccess(nodeId utils.NodeId, buffer []byte, tasks ...task.Task) {
	logs.Initialize()
	if len(tasks) > 0 {
		s.ReceiveTaskSuccess(nodeId, buffer, tasks[0])
	} else {
		logs.Log.Info("[INFO] Success")
	}
}

func (s *System) ParseResult(resString string) (float64, error) {
	logs.Initialize()
	// Split the string to isolate the number
	number, err := strconv.ParseFloat(resString, 64)
	if err != nil {
		logs.Log.WithField("Error parsing float", err).Error("[ERROR]")
		return 0.0, fmt.Errorf("error parsing float: %v", err)
	}
	return number, nil
}

func (s *System) ReceiveTaskSuccess(nodeId utils.NodeId, buffer []byte, task task.Task) {
	logs.Initialize()
	if !task.IsFinished {
		logs.Log.WithFields(logrus.Fields{
			"taskID":  task.Id,
			"process": task.IdProc,
			"node":    nodeId,
		}).Infof("[INFO] Task %v (Process %v) successfully assigned to Node %v.\n", task.Id, task.IdProc, nodeId)
	} else {
		logs.Log.WithFields(logrus.Fields{
			"taskID":  task.Id,
			"process": task.IdProc,
			"node":    nodeId,
		}).Infof("[WARNING] Task %v (Process %v) was reported as already finished by Node %v. Ignoring...\n", task.Id, task.IdProc, nodeId)
	}
}

func (s *System) ReceiveReturnedRes(task task.Task, nodeId utils.NodeId, resString string, conn net.Conn) {
	res, err := s.ParseResult(resString)
	if err != nil {
		task.IsFinished = false
		msgFailure := message.NewMessage(message.ActionFailure, fmt.Sprintf("Failed to update procedure: %v: %v", task.IdProc, err), task, 0)
		logs.Log.WithFields(logrus.Fields{
			"ProcedureID": task.IdProc,
			"error":       err,
		}).Errorf("[ERROR] Failed to update procedure: %v: %v", task.IdProc, err)
		message.SendMessage(msgFailure, conn)
	}
	s.taskScheduler.HandleProcessUpdate(task, nodeId, res)
}

// StartHeartbeatChecker periodically checks node health.
func (s *System) StartHeartbeatChecker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.CheckHeartbeat()
	}
}

func (s *System) CheckHeartbeat() {
	logs.Initialize()
	s.mu.Lock()
	s.muConn.Lock()
	defer s.mu.Unlock()
	defer s.muConn.Unlock()

	for nodeId, checks := range s.healthRegistry {
		newChecks := checks + 1
		s.healthRegistry[nodeId] = newChecks

		if newChecks >= 3 {
			logs.Log.WithField("Node is unresponsive. Reassigning tasks and removing from system. Node id", nodeId).Warning("[WARNING]")
			s.taskScheduler.ReassignTasksForNode(nodeId)
			delete(s.healthRegistry, nodeId)
			delete(s.systemNodes, nodeId)
		}
	}
}

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

func (s *System) RemoveSystemNode(nodeId utils.NodeId) {
	logs.Initialize()
	s.muConn.Lock()
	defer s.muConn.Unlock()
	logs.Log.WithField("Removing Node %v from the system, Node ID", nodeId).Info("[INFO]")
	delete(s.systemNodes, nodeId)
	logs.Log.WithFields(logrus.Fields{
		"NodeID": nodeId,
		"lenght": len(s.systemNodes),
	}).Infof("[INFO] Node %v successfully removed. Remaining nodes: %d\n", nodeId, len(s.systemNodes))

}

// GetConnection safely retrieves a connection by utils.NodeId.
func (s *System) GetConnection(nodeId utils.NodeId) (net.Conn, bool) {
	s.muConn.RLock()
	defer s.muConn.RUnlock()
	conn, exists := s.systemNodes[nodeId]
	return conn, exists
}

func (s *System) CreateNewProcess(entryArray []float64) {
	logs.Initialize()
	s.muConn.Lock()
	defer s.muConn.Unlock()
	if len(s.systemNodes) == 0 {
		logs.Log.Error("[ERROR] No nodes available to handle the process.")
		return
	}
	logs.Log.Info("[INFO] Creating a new process for entry array", entryArray)
	processId, err := s.taskScheduler.CreateNewProcess(entryArray, len(s.systemNodes), s.systemNodes)
	if err != nil {
		logs.Log.Error("[ERROR] Failed to create tasks for process", err)
		return
	}
	logs.Log.Info("[INFO] Successfully created process with id", processId)
}

func (s *System) WaitlistRetry() {
	for {
		time.Sleep(1 * time.Second)
		s.muConn.Lock()
		s.taskScheduler.AttemptReasign(s.systemNodes)
		s.muConn.Unlock()
	}
}

func (s *System) TaskScheduler() *tscheduler.TScheduler {
	return &s.taskScheduler
}
