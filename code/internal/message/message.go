package message

import (
	"distributed-system/internal/task"
	"encoding/json"
	"fmt"
	"net"
)

type ActionType string

const (
	Heartbeat     ActionType = "hearbeat"
	RemoveTask    ActionType = "removeTask"
	ReturnedRes   ActionType = "returnedRes"
	AsignTask     ActionType = "asignTask"
	RejectAsign   ActionType = "rejectAsign"
	NotifyNodeUp  ActionType = "nodeUp"
	CleanUp       ActionType = "cleanUp"
	ActionSuccess ActionType = "success"
	ActionFailure ActionType = "failure"
)

type Message struct {
	Action           ActionType `json:"action"`
	Content          string
	Task             task.Task
	Sender           int      `json:"idNode"`
	ConnectionHandle net.Conn `json:"ConnectionHandle"`
}

func NewMessage(action ActionType, content string, task task.Task, sender int) *Message {
	return &Message{
		Action:  action,
		Content: content,
		Task:    task,
		Sender:  sender,
	}
}

func NewMessageNoTask(action ActionType, content string, sender int) *Message {
	return &Message{
		Action:  action,
		Content: content,
		Task:    task.Task{},
		Sender:  sender,
	}
}

func SendMessage(msg Message, conn net.Conn) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("error sending data: %w", err)
	}
	return nil
}

func InterpretMessage(buffer []byte, size int) (*Message, error) {
	var msg Message
	err := json.Unmarshal(buffer[:size], &msg)
	if err != nil {
		return &Message{}, fmt.Errorf("error unmarshaling: %v", err)
	}
	return &msg, nil
}
