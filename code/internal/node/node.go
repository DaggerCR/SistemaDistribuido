package node

import (
	"distributed-system/internal/message"
	"distributed-system/internal/task"
	"distributed-system/pkg/customerrors"
	"distributed-system/pkg/utils"
	"errors"
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

func (n *Node) DeleteTask(id utils.TaskId) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.taskQueue) == 0 {
		return
	}
	newQueue := n.taskQueue[:0]
	for _, v := range n.taskQueue {
		if v.Id != id {
			newQueue = append(newQueue, v)
		}
	}
	n.taskQueue = newQueue
}

func (n *Node) PopTask(idx int) (task.Task, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	deletedTask, isSafe := task.SafeTaskAccess(n.taskQueue, idx)
	if !isSafe {
		return task.Task{}, errors.New("no task found for pop action")
	}
	return deletedTask, nil
}

func (n *Node) calcSum(idx int) float64 {
	n.mu.Lock()
	defer n.mu.Unlock()
	var sum float64
	chunk := n.taskQueue[idx].Chunk
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
	fmt.Printf("Conexion de tipo: %v, en el puerto %v\n", protocol, address)
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
	fmt.Printf("Me, node %v, recieved content: %v\n", n.id, msg.Content)
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

func (n *Node) ReceiveAsignTask(task task.Task, nodeId utils.NodeId) {
	n.AppendTask(task)
	content := fmt.Sprintf("Task asigned: %v\n", task.Id)
	fmt.Println("Taks was asigned; overview of tasks: ", task)
	nodeUpMsg := message.NewMessage(message.ActionSuccess, content, task, n.id)
	message.SendMessage(nodeUpMsg, n.conn)
}
