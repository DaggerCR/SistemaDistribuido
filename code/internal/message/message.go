package message

import "distributed-system/internal/task"

type ActionType string

const (
	heartbeat    ActionType = "hearbeat"
	removeTask   ActionType = "removeTask"
	unlockTask   ActionType = "unlockTask"
	returnedRes  ActionType = "returnedRes"
	asignTask    ActionType = "asignTask"
	rejectAsign  ActionType = "rejectAsign"
	notifyNodeUp ActionType = "nodeUp"
)

type Message struct {
	action  ActionType
	content string
	task    task.Task
}

func (m *Message) NewMessage() *Message {
	return &Message{}
}

// Getters

// Action returns the action in the Message.
func (m *Message) Action() ActionType {
	return m.action
}

// Task returns the task in the Message.
func (m *Message) Task() task.Task {
	return m.task
}

// Setters

// SetAction sets the action in the Message.
func (m *Message) SetAction(action ActionType) {
	m.action = action
}

// SetTask sets the task in the Message.
func (m *Message) SetTask(task task.Task) {
	m.task = task
}
