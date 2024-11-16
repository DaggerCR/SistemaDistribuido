package tscheduler

import (
	"distributed-system/internal/task"
	"distributed-system/pkg/utils"
	"fmt"
)

type NodeId int
type Load int

type TScheduler struct {
	taskRegistry []task.Task //data redundancy (to manage task reasignment )
	taskQueue    []task.Task //queue for unasigned task (aka task that weren't able to fit due to load balancing or max load of all nodes reached)
}

// NewTScheduler initializes and returns a new TScheduler instance.
func NewTScheduler() *TScheduler {
	return &TScheduler{}
}

// Getters

// TaskRegistry returns the slice of tasks in the taskRegistry.
func (ts *TScheduler) TaskRegistry() []task.Task {
	return ts.taskRegistry
}

// TaskQueue returns the slice of tasks in the taskQueue.
func (ts *TScheduler) TaskQueue() []task.Task {
	return ts.taskQueue
}

// Setters

// SetTaskRegistry updates the taskRegistry slice.
func (ts *TScheduler) SetTaskRegistry(tasks []task.Task) {
	ts.taskRegistry = tasks
}

// SetTaskQueue updates the taskQueue slice.
func (ts *TScheduler) SetTaskQueue(tasks []task.Task) {
	ts.taskQueue = tasks
}

// Append Methods

// AppendToTaskRegistry adds a single task to the taskRegistry slice.
func (ts *TScheduler) AppendToTaskRegistry(task task.Task) {
	ts.taskRegistry = append(ts.taskRegistry, task)
}

// AppendToTaskQueue adds a single task to the taskQueue slice.
func (ts *TScheduler) AppendToTaskQueue(task task.Task) {
	ts.taskQueue = append(ts.taskQueue, task)
}

func (ts *TScheduler) CreateTasks(entryArray []float64, numNodes int, idProc int) ([]task.Task, int, error) {
	//numNodes never passes as less than or equal 0
	chunkLen := len(entryArray) / numNodes
	var tasks []task.Task
	chunks, err := utils.SliceUpArray(entryArray, chunkLen)
	if err != nil {
		return nil, 0, fmt.Errorf("could not create task for procedure in this moment: %w", err)
	}
	for idx, chunk := range chunks {
		newTask := task.NewTask(idProc+idx, idProc, chunk)
		tasks = append(tasks, *newTask)
	}
	return tasks, len(tasks), nil
}

/*
func (ts *TScheduler) asignTasks(tasks []task.Task, sys *System) {
	idx := 0
	sys.mu.RLock()

	for _, val := range systemNodes {
		content := fmt.Sprintf("Asigned task with id %v from process: %v", tasks[idx].Id, tasks[idx].IdProc)
		msg := message.NewMessage(message.AsignTask, content, tasks[idx], 0)
		if err := message.SendMessage(*msg, val); err != nil {
			ts.AppendToTaskQueue(tasks[idx])
			idx++
			continue
		}
	}

	sys.mu.RUnlock()
	if len(tasks) > len(sys.systemNodes) {
		lessLoadedNodes := sys.ProbeLessLoadedNodes()

		//if num of tasks is greater than num of nodes, asign rest of tasks between less loaded nodes
		for nodeId := range lessLoadedNodes {
		}
	}

}
*/
