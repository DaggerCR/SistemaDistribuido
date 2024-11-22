package node

import (
	"distributed-system/internal/message"
	"distributed-system/internal/task"
	"distributed-system/logs"
	"distributed-system/pkg/customerrors"
	"distributed-system/pkg/utils"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Node struct {
	id        int
	taskQueue []task.Task
	conn      net.Conn
	mu        sync.Mutex
}

func NewNode(id int) (*Node, error) {
	logs.Initialize()
	maxSize, err := strconv.Atoi(os.Getenv("MAX_TASKS"))
	if err != nil {
		logs.Log.WithField("error creating a new node", err).Error("An error ocurred while creating nodes")
		return nil, fmt.Errorf("error creating an element %w", err)
	}

	return &Node{
		id:        id,
		taskQueue: make([]task.Task, 0, maxSize),
	}, nil
}

func (n *Node) Id() int {
	return n.id
}

func (n *Node) TaskQueue() []task.Task {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.taskQueue
}

func (n *Node) Conn() net.Conn {
	return n.conn
}

func (n *Node) AppendTask(newTask task.Task) error {
	logs.Initialize()
	n.mu.Lock()
	defer n.mu.Unlock()
	n.taskQueue = append(n.taskQueue, newTask)
	logs.Log.WithFields(logrus.Fields{
		"taskId":    newTask.Id,
		"NodeID":    n.id,
		"QueueSize": len(n.taskQueue),
	}).Infof("[DEBUG] Appended task %v, from Node %v, with queue size is %v", newTask.Id, n.id, len(n.taskQueue))
	return nil
}

func (n *Node) PopTask() (task.Task, bool) {
	logs.Initialize()
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.taskQueue) > 0 {
		task := n.taskQueue[0]
		newQueue := n.taskQueue[1:]
		n.taskQueue = newQueue
		logs.Log.WithFields(logrus.Fields{
			"taskId":    task.Id,
			"NodeID":    n.id,
			"QueueSize": len(n.taskQueue),
		}).Infof("[DEBUG] Popped task %v, from Node %v, Remaining queue size is %v", task.Id, n.id, len(n.taskQueue))

		return task, true
	}
	return *task.NewTask(-1, -1, []float64{}), false
}

func (n *Node) CalcSum(task task.Task) float64 {
	logs.Initialize()
	n.mu.Lock()
	defer n.mu.Unlock()
	var sum float64
	chunk := task.Chunk
	for _, val := range chunk {
		sum += val
	}
	logs.Log.WithField("The total sum is", sum).Info("Result of the sum from the task", task.Print())
	return sum
}

func (n *Node) Connect(protocol string, address string) error {
	logs.Initialize()
	conn, err := net.Dial(protocol, address)
	if err != nil {
		logs.Log.WithField("error connecting to a node", err).Error("An error ocurred while connecting nodes")
		return fmt.Errorf("error connecting node: %w", err)
	}
	n.conn = conn
	return nil
}

func Panic(protocol string, address string, id int, errorS error) {
	logs.Initialize()
	conn, err := net.Dial(protocol, address)
	if err != nil {
		customerrors.HandleError(err)
		logs.Log.WithField("Pacnic error", err).Error("A panic error in node.go")
		return
	}
	defer conn.Close()
	content := fmt.Sprintf("Failed to create or connect node, panic! : %v", errorS)
	msg := message.NewMessageNoTask(message.ActionFailure, content, id)
	logs.Log.WithField(string(message.ActionFailure), content).Info("ID", id)
	err = message.SendMessage(msg, conn)
	if err != nil {
		logs.Log.WithField("Pacnic error", err).Error("A panic error in node.go")
		customerrors.HandleError(err)
	}
}

func SetUpConnection(protocol string, address string, id int) (*Node, error) {
	logs.Initialize()
	nnode, err := NewNode(id)
	if err != nil {
		Panic(protocol, address, id, err)
		os.Exit(1)
	}
	err = nnode.Connect(protocol, address)
	if err != nil {
		Panic(protocol, address, id, err)
		os.Exit(1)
	} else {
		msg := message.NewMessageNoTask(message.NotifyNodeUp, "Node created succesfully", id)
		logs.Log.WithField("Node created succesfully", id).Info(message.NotifyNodeUp)
		err = message.SendMessage(msg, nnode.Conn())
	}
	if err != nil {
		logs.Log.WithField("error seting connection up", err).Error("An error while trying to set up conection")
		return &Node{}, fmt.Errorf("error seting connection up: %w", err)
	}
	return nnode, nil
}

func (n *Node) HandleNodeConnection() error {
	logs.Initialize()
	logs.Log.WithField("the Node is starting connection handler", n.id).Info("[INFO]")
	go n.sendHeartBeat()
	go n.ExecuteTask()

	for {
		buffer, err := message.RecieveMessage(n.conn)
		if err != nil {
			if err.Error() == "EOF" {
				logs.Log.WithField("Connection lost for Node", n.id).Error("[ERROR]")
				return fmt.Errorf("connection lost for node %v: %w", n.id, err)
			} else {
				logs.Log.WithField("Failed to read from connection for Node", n.id).Error("[ERROR]")
				logs.Log.WithField("error", err).Info("continue: ")
				fmt.Printf("error, check log")
			}
		}
		logs.Log.WithField("Node received a message", n.id).Debug("[DEBUG]")
		go n.HandleReceivedData(buffer)
	}
}

func (n *Node) sendHeartBeat() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		msg := message.NewMessageNoTask(message.Heartbeat, "Heartbeat", n.id)
		err := message.SendMessage(msg, n.conn)
		if err != nil {
			logs.Log.WithField("Error sending heartBeat up", err).Error("[ERROR]")
			return
		}
	}
}

func (n *Node) HandleReceivedData(buffer []byte) {
	logs.Initialize()
	msg, err := message.InterpretMessage(buffer)
	if err != nil {
		logs.Log.WithField("Invalid message recieved", n.id).Error("Error handle data received")
		errorMsg := message.NewMessageNoTask(message.ActionFailure, "Invalid message revieved", n.id)
		message.SendMessage(errorMsg, n.conn)
	}
	logs.Log.WithField("Me, node", n.id).Info("Content recieved correctly")
	logs.Log.WithField("content", msg.Content).Info("Content recieved correctly")
	switch msg.Action {
	case message.AsignTask:
		n.ReceiveAsignTask(msg.Task, utils.NodeId(msg.Sender))
	default:
		logs.Log.WithField("Not implemented yet", msg.Content).Info("Error of implementation")
	}
}

func (n *Node) ExecuteTask() {
	logs.Initialize()
	for {
		time.Sleep(2 * time.Second)
		taskToExecute, ok := n.PopTask()
		isFinished := taskToExecute.IsFinished
		if ok && !isFinished {
			res := n.CalcSum(taskToExecute)

			logs.Log.WithFields(logrus.Fields{
				"TaskToExecute": taskToExecute.Print(),
				"Result":        res,
			}).Infof("[ %v executed with result: %v ]", taskToExecute.Print(), res)

			resString := strconv.FormatFloat(res, 'f', -1, 64)
			taskToExecute.IsFinished = true
			msg := message.NewMessage(message.ReturnedRes, resString, taskToExecute, n.id)
			logs.Log.WithFields(logrus.Fields{
				"Message":      message.ReturnedRes,
				"resString":    resString,
				"TasktoExcute": taskToExecute,
				"ID":           n.id,
			}).Infof("Message %v, with string %v, the task was %v and the Node is %v", message.ReturnedRes, resString, taskToExecute, n.id)
			message.SendMessage(msg, n.Conn())
		}
	}
}

func (n *Node) ReceiveAsignTask(task task.Task, nodeId utils.NodeId) {
	logs.Initialize()
	n.AppendTask(task)
	content := fmt.Sprintf("Task asigned: %v\n", task.Id)
	logs.Log.WithField("Taks was asigned; overview of tasks", task.Print())
	n.Print()
	nodeUpMsg := message.NewMessage(message.ActionSuccess, content, task, n.id)
	logs.Log.WithFields(logrus.Fields{
		"ActionSuccess": message.ActionSuccess,
		"Connet":        content,
		"Task":          task,
		"ID":            n.id,
	}).Infof("Status %v, with content %v, the task was %v and the Node is %v", message.ActionSuccess, content, task, n.id)
	message.SendMessage(nodeUpMsg, n.conn)
}

// this function save on the log everything
func (n *Node) Print() {
	n.mu.Lock()
	defer n.mu.Unlock()
	logs.Log.WithField("----------Node with id", n.id)
	for idx, task := range n.taskQueue {
		fmt.Printf("%v:\t%v\n", idx, task.Print())
		logs.Log.WithFields(logrus.Fields{
			"idx":       idx,
			"taskPrint": task.Print(),
		}).Infof("%v:\t%v\n", idx, task.Print())
	}
}
