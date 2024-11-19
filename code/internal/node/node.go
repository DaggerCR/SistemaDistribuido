package node

import (
	"distributed-system/internal/message"
	"distributed-system/internal/task"
	"distributed-system/pkg/customerrors"
	"distributed-system/pkg/utils"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type Node struct {
	id        int
	taskQueue []task.Task
	conn      net.Conn
	mu        sync.Mutex
}

func NewNode(id int) (*Node, error) {
	maxSize, err := strconv.Atoi(os.Getenv("MAX_TASKS"))
	if err != nil {
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

func (n *Node) SetID(id int) {
	n.id = id
}

func (n *Node) SetTaskQueue(taskQueue []task.Task) {
	n.taskQueue = taskQueue
}

func (n *Node) AppendTask(newTask task.Task) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.taskQueue = append(n.taskQueue, newTask)
}

func (n *Node) DeleteTassByIndex(idx int) (task.Task, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.taskQueue) == 0 {
		return *task.NewTask(-1, -1, []float64{}), false
	}
	if idx < 0 || idx >= len(n.taskQueue) {
		return *task.NewTask(-1, -1, []float64{}), false
	}
	deletedTask := n.taskQueue[idx]
	newQueue := append(n.taskQueue[:idx], n.taskQueue[idx+1:]...)
	n.taskQueue = newQueue
	return deletedTask, true
}

func (n *Node) PopTask() (task.Task, bool) {
	return n.DeleteTassByIndex(0)
}

func (n *Node) CalcSum(task task.Task) float64 {
	n.mu.Lock()
	defer n.mu.Unlock()
	var sum float64
	chunk := task.Chunk
	for _, val := range chunk {
		sum += val
	}
	return sum
}

func (n *Node) Connect(protocol string, address string) error {
	conn, err := net.Dial(protocol, address)
	if err != nil {
		return fmt.Errorf("error connecting node: %w", err)
	}
	n.conn = conn
	return nil
}

func Panic(protocol string, address string, id int, errorS error) {
	conn, err := net.Dial(protocol, address)
	if err != nil {
		customerrors.HandleError(err)
		return
	}
	defer conn.Close()
	content := fmt.Sprintf("Failed to create or connect node, panic! : %v", errorS)
	msg := message.NewMessageNoTask(message.ActionFailure, content, id)
	err = message.SendMessage(msg, conn)
	if err != nil {
		customerrors.HandleError(err)
	}
}

func SetUpConnection(protocol string, address string, id int) (*Node, error) {
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
		err = message.SendMessage(msg, nnode.Conn())
	}
	if err != nil {
		return &Node{}, fmt.Errorf("error seting connection up: %w", err)
	}
	return nnode, nil
}

func (n *Node) HandleNodeConnection() error {
	go n.sendHeartBeat()
	go n.ExecuteTask()
	for {
		buffer, err := message.RecieveMessage(n.conn)
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Conection with System port lost") //cuando se cierra el servidor aparece este mensaje
			} else {
				fmt.Println("Error reading from client...:", err)
			}
			msg := message.NewMessageNoTask(message.ActionFailure, "Failed to read from node buffer", n.id)
			err = message.SendMessage(msg, n.conn)
			if err != nil {
				return fmt.Errorf("error handling connection up: %w", err)
			}
		}
		go n.HandleReceivedData(buffer)
	}
}

func (n *Node) sendHeartBeat() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		msg := message.NewMessageNoTask(message.Heartbeat, "Heartbeat", n.id)
		err := message.SendMessage(msg, n.conn)
		if err != nil {
			fmt.Println("Error sending heartBeat up: %w", err)
			return
		}
	}
}

func (n *Node) HandleReceivedData(buffer []byte) {
	msg, err := message.InterpretMessage(buffer)
	if err != nil {
		errorMsg := message.NewMessageNoTask(message.ActionFailure, "Invalid message revieved", n.id)
		message.SendMessage(errorMsg, n.conn)
	}
	//fmt.Printf("Me, node %v, recieved content: %v\n", n.id, msg.Content)
	switch msg.Action {
	case message.AsignTask:
		n.ReceiveAsignTask(msg.Task, utils.NodeId(msg.Sender))

	case message.CleanUp:
		n.SetTaskQueue([]task.Task{})
		nodeUpMsg := message.NewMessageNoTask(message.ActionSuccess, "CleanUp completed", n.id)
		message.SendMessage(nodeUpMsg, n.conn)
	default:
		fmt.Println("Not implemented yet", msg.Content, "*")
	}
}

func (n *Node) ExecuteTask() {
	for {
		time.Sleep(1 * time.Millisecond)
		taskToExecute, ok := n.PopTask()
		isFinished := taskToExecute.IsFinished
		if ok && !isFinished {
			res := n.CalcSum(taskToExecute)
			fmt.Println("")
			fmt.Println("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
			fmt.Printf("\n [ %v executed with result: %v ]\n", taskToExecute.Print(), res)
			fmt.Println("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
			fmt.Println("")
			resString := strconv.FormatFloat(res, 'f', -1, 64)
			taskToExecute.IsFinished = true
			msg := message.NewMessage(message.ReturnedRes, resString, taskToExecute, n.id)
			message.SendMessage(msg, n.Conn())
		}
	}
}

func (n *Node) ReceiveAsignTask(task task.Task, nodeId utils.NodeId) {
	n.AppendTask(task)
	content := fmt.Sprintf("Task asigned: %v\n", task.Id)
	fmt.Printf("\nTaks was asigned; overview of tasks: %v \n", task.Print())
	n.Print()
	nodeUpMsg := message.NewMessage(message.ActionSuccess, content, task, n.id)
	message.SendMessage(nodeUpMsg, n.conn)
}

func (n *Node) Print() {
	n.mu.Lock()
	defer n.mu.Unlock()
	fmt.Printf("\n\n")
	fmt.Printf("====================================== Node with id %v ======================================", n.id)
	fmt.Println("")
	for idx, task := range n.taskQueue {
		fmt.Println("---------------------------------------------------------------------------------------------")
		fmt.Printf("%v:\t%v\n", idx, task.Print())
	}
	fmt.Println("---------------------------------------------------------------------------------------------")
	fmt.Println("\n")
}
