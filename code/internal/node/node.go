package node

import (
	"distributed-system/internal/task"
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
	isActive  bool
	conn      net.Conn
}

func NewNode(id int) (*Node, error) {
	maxSize, err := utils.ParseStringUint8(os.Getenv("MAX_SIZE"))
	if err != nil {
		fmt.Println("Error with node")
		return nil, fmt.Errorf("error creating an element %w", err)
	}
	fmt.Println("Node created succesfully")
	return &Node{
		id:        id,
		taskQueue: make([]task.Task, maxSize),
	}, nil
}

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

	//Send to server initial messages
	initialMessage := strconv.Itoa(n.id) + "-nodeUp"
	n.SendMessageToSystem(conn, initialMessage)
	n.ReceiveMessageFromSystem(conn)
	n.ReturnHeartbeat(conn)
}

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

func (n *Node) Id() int {
	return n.id
}

func (n *Node) IsActive() bool {
	return n.isActive
}

func (n *Node) TaskQueue() []task.Task {
	return n.taskQueue
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

func (n *Node) PopTask(idx int) (task.Task, error) {
	deletedTask, isSafe := task.SafeTaskAccess(n.taskQueue, idx)
	if !isSafe {
		return task.Task{}, errors.New("no task found for pop action")
	}
	return deletedTask, nil
}

func (n *Node) calcSum(idx int) float64 {
	var sum float64
	chunk := n.taskQueue[idx].Chunk()
	for _, val := range chunk {
		sum += val
	}
	return sum
}

func Connect() {

}

//todo: calc for std deviation and mean
