package tscheduler

import "distributed-system/internal/task"

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
