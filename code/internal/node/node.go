package node

import (
	"distributed-system/internal/task"
	"distributed-system/pkg/utils"
	"errors"
	"fmt"
	"os"
)

type Node struct {
	id        int
	taskQueue []task.Task
	isActive  bool
}

func NewNode(id int) (*Node, error) {
	maxSize, err := utils.ParseStringUint8(os.Getenv("MAX_SIZE"))
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
