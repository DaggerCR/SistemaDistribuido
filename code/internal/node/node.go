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
	"time"
)

type Node struct {
	id        int
	taskQueue []task.Task
	//messageQueue []message.Message
	isActive bool
	conn     net.Conn
}

func NewNode(id int) (*Node, error) {
	maxSize, err := strconv.Atoi(os.Getenv("MAX_TASKS"))
	if err != nil {
		return nil, fmt.Errorf("error creating an element %w", err)
	}

	return &Node{
		id:        id,
		taskQueue: make([]task.Task, maxSize),
	}, nil
}

func (n *Node) Id() int {
	return n.id
}

func (n *Node) IsActive() bool {
	return n.isActive
}

func (n *Node) TaskQueue() []task.Task {
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

func (n *Node) SetActive(isActive bool) {
	n.isActive = isActive
}

func (n *Node) AppendTask(newTask task.Task) {
	n.taskQueue = append(n.taskQueue, newTask)
}

func (n *Node) DeleteTask(id utils.TaskId) {
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
	deletedTask, isSafe := task.SafeTaskAccess(n.taskQueue, idx)
	if !isSafe {
		return task.Task{}, errors.New("no task found for pop action")
	}
	return deletedTask, nil
}

func (n *Node) calcSum(idx int) float64 {
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
	err = message.SendMessage(*msg, conn)
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
		err = message.SendMessage(*msg, nnode.Conn())
	}
	if err != nil {
		return &Node{}, fmt.Errorf("error seting connection up: %w", err)
	}
	return nnode, nil
}

func (n *Node) HandleNodeConnection() error {
	var messageQueue [][]byte
	go n.sendHeartBeat()

	for {
		rawMsg := make([]byte, 1024)
		size, err := n.conn.Read(rawMsg)
		messageQueue := append(messageQueue, rawMsg)
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Conection with System port lost") //cuando se cierra el servidor aparece este mensaje
			} else {
				fmt.Println("Error reading from client...:", err)
			}
			msg := message.NewMessageNoTask(message.ActionFailure, "Failed to read from node buffer", n.id)
			err = message.SendMessage(*msg, n.conn)
			if err != nil {
				return fmt.Errorf("error handling connection up: %w", err)
			}
		}
		go n.HandleReceivedData(messageQueue[len(messageQueue)-1][:size], size)
	}
}

func (n *Node) sendHeartBeat() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		msg := message.NewMessageNoTask(message.Heartbeat, "Heartbeat", n.id)
		err := message.SendMessage(*msg, n.conn)
		if err != nil {
			fmt.Println("Error sending heartBeat up: %w", err)
			return
		}
	}
}

func (n *Node) HandleReceivedData(buffer []byte, size int) {
	fmt.Printf("Me, node %v, recieved message from master\n", n.id)
	msg, err := message.InterpretMessage(buffer, size)
	if err != nil {
		errorMsg := message.NewMessageNoTask(message.ActionFailure, "Invalid message revieved", n.id)
		message.SendMessage(*errorMsg, n.conn)
	}
	fmt.Printf("Me, node %v, recieved content: %v\n", n.id, msg.Content)
	switch msg.Action {
	case message.AsignTask:
		n.AppendTask(msg.Task)
		content := fmt.Sprintf("Task asigned: %v\n", msg.Task.Id)
		nodeUpMsg := message.NewMessage(message.ActionSuccess, content, msg.Task, n.id)
		message.SendMessage(*nodeUpMsg, n.conn)
		fmt.Printf("Task chunk is: %v \n", msg.Task.Chunk)
	case message.CleanUp:
		n.SetTaskQueue([]task.Task{})
		nodeUpMsg := message.NewMessage(message.ActionSuccess, "CleanUp completed", msg.Task, n.id)
		message.SendMessage(*nodeUpMsg, n.conn)
	default:
		fmt.Println("Not implemented yet")
	}
}

//
//
//

//JOSI

func (n *Node) ConnectToSystem() {
	utils.LoadVEnv()
	protocol := os.Getenv("PROTOCOL")
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")
	address := fmt.Sprintf("%s:%s", host, port)
	conn, err := net.Dial(protocol, address)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("Node %d connected to server succesfully\n", n.id)
	n.conn = conn
	n.ReceiveMessageFromSystem(conn)
	n.ReturnHeartbeat(conn)
}

//Revisar esto, MUST CHANGE

func (n *Node) SendMessageToSystem(conn net.Conn, message string) {
	_, err := conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}
}

func (n *Node) ReceiveMessageFromSystem(conn net.Conn) {
	buf := make([]byte, 1024)
	num, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}
	fmt.Printf("Me, node %d , received from server: %s \n", n.id, string(buf[:num]))
}

func (n *Node) ReturnHeartbeat(conn net.Conn) {
	for {
		//Node will send msj to system as 'heatbeat'
		message := strconv.Itoa(n.id) + "-heartbeat"
		n.SendMessageToSystem(conn, message)
		time.Sleep(3 * time.Second)
	}
}

func (n *Node) ReturnHeartbeat2() {
	for {
		//Node will send msj to system as 'heatbeat'
		msg := message.NewMessageNoTask(message.Heartbeat, "", n.id)
		message.SendMessage(*msg, n.conn)
		time.Sleep(3 * time.Second)
	}
}

//todo: calc for std deviation and mean
