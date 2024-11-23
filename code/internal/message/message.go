package message

import (
	"distributed-system/internal/task"
	"distributed-system/logs"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
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
	Action  ActionType
	Content string
	Task    task.Task
	Sender  int
	mu      sync.Mutex
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
		Task:    *task.NewTask(-1, -1, []float64{}),
		Sender:  sender,
	}
}

func SendMessage(msg *Message, conn net.Conn) error {
	logs.Initialize()
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}
	messageLenght := uint32(len(data))
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, messageLenght)
	data = append(header, data...)
	msg.mu.Lock()
	defer msg.mu.Unlock()
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("error sending data: %w", err)
	}
	return nil
}

func InterpretMessage(buffer []byte) (*Message, error) {
	logs.Initialize()
	var msg Message
	//strip 4 bytes from header size
	if len(buffer) < 4 {
		return &Message{}, errors.New("size of message too small")
	}
	err := json.Unmarshal(buffer, &msg)
	if err != nil {
		return &Message{}, fmt.Errorf("error unmarshaling: %v", err)
	}
	return &msg, nil
}

func RecieveMessage(conn net.Conn) ([]byte, error) {
	logs.Initialize()
	header := make([]byte, 4)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, fmt.Errorf("error reading header :%v", err)
	}
	messageLenght := binary.BigEndian.Uint32(header)
	message := make([]byte, messageLenght)
	_, err = io.ReadFull(conn, message)
	if err != nil {
		return nil, fmt.Errorf("error reading message: %v", err)
	}
	return message, nil
}
