package tscheduler

import (
	"distributed-system/internal/message"
	"distributed-system/internal/system"

	"distributed-system/internal/task"
	"distributed-system/pkg/utils"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
)

type NodeId int
type Load int

type TScheduler struct {
	taskRegistry []task.Task //data redundancy (to manage task reasignment )
	taskWaitlist []task.Task //queue for unasigned task (aka task that weren't able to fit due to load balancing or max load of all nodes reached)

	maxNodeLoad int
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

// TaskQueue returns the slice of tasks in the taskWaitlist.
func (ts *TScheduler) TaskQueue() []task.Task {
	return ts.taskWaitlist
}

// Setters

// SetTaskRegistry updates the taskRegistry slice.
func (ts *TScheduler) SetTaskRegistry(tasks []task.Task) {
	ts.taskRegistry = tasks
}

// SetTaskQueue updates the taskWaitlist slice.
func (ts *TScheduler) SetTaskQueue(tasks []task.Task) {
	ts.taskWaitlist = tasks
}

// Append Methods

// AppendToTaskRegistry adds a single task to the taskRegistry slice.
func (ts *TScheduler) AppendToTaskRegistry(task task.Task) {
	ts.taskRegistry = append(ts.taskRegistry, task)
}

// AppendToTaskQueue adds a single task to the taskWaitlist slice.
func (ts *TScheduler) AppendToTaskQueue(task task.Task) {
	ts.taskWaitlist = append(ts.taskWaitlist, task)
}

func (ts *TScheduler) CreateTasks(entryArray []float64, numNodes int, idProc int) ([]task.Task, int, error) {
	//numNodes never passes as less than or equal 0
	chunkLen := int(math.Ceil(float64(len(entryArray)) / float64(numNodes)))
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

func (ts *TScheduler) asignTasks(tasks []task.Task, connections map[NodeId]net.Conn, sys *system.System) {
	idx := 0 // Tracks the number of tasks assigned so far

	// Retrieve the maximum allowed load per node from the environment or use a default
	maxNodeLoad, err := strconv.Atoi(os.Getenv("MAX_TASKS"))
	if err != nil {
		maxNodeLoad = system.DefaultMaxLoad
	}

	// Step 1: Attempt to assign tasks to available nodes within their load capacity
	// Only nodes with load < maxNodeLoad are considered
	if len(tasks) >= len(connections) { // More tasks than nodes
		for nodeId, conn := range connections {
			if idx >= len(tasks) {
				break // All tasks have been assigned
			}

			// Check the current load of the node
			load, found := sys.GetLoadBalance(system.NodeId(nodeId))
			if found && load >= system.Load(maxNodeLoad) {
				continue // Skip nodes that are already at or above max load
			}

			// Create and send the task assignment message
			content := fmt.Sprintf("Assigned task with id %v from process: %v", tasks[idx].Id, tasks[idx].IdProc)
			msg := message.NewMessage(message.AsignTask, content, tasks[idx], 0)
			if err := message.SendMessage(*msg, conn); err != nil {
				// If sending fails, requeue the task for later assignment
				ts.AppendToTaskQueue(tasks[idx])
				continue
			}

			idx++ // Successfully assigned this task
		}
	}

	// Step 2: Distribute remaining tasks among the least-loaded nodes
	// This block is entered only if there are pending tasks
	if idx < len(tasks) {
		// Identify nodes with the least load to handle remaining tasks
		lessLoadedNodes := sys.ProbeLessLoadedNodes(len(tasks) - idx)
		for _, nodeId := range lessLoadedNodes {
			if idx >= len(tasks) {
				break // All tasks have been assigned
			}

			// Create and send the task assignment message
			content := fmt.Sprintf("Assigned task with id %v from process: %v", tasks[idx].Id, tasks[idx].IdProc)
			msg := message.NewMessage(message.AsignTask, content, tasks[idx], 0)
			if err := message.SendMessage(*msg, connections[NodeId(nodeId)]); err != nil {
				// If sending fails, requeue the task for later assignment
				ts.AppendToTaskQueue(tasks[idx])
				continue
			}

			idx++ // Successfully assigned this task
		}
	}

	// Step 3: Handle tasks that could not be assigned
	// Remaining tasks (if any) are appended to the task waitlist
	if idx < len(tasks) {
		ts.taskWaitlist = append(ts.taskWaitlist, tasks[idx:]...)
	}
}
