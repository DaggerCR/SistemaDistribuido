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
		logs.Log.WithError(err).Error("Error reading message from json")
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
		logs.Log.WithError(err).Error("Error sending message")
		return fmt.Errorf("error sending data: %w", err)
	}
	logs.Log.WithField("Message lenght", messageLenght).Info("Message send without errors")
	return nil
}

func InterpretMessage(buffer []byte) (*Message, error) {
	logs.Initialize()
	var msg Message
	//strip 4 bytes from header size
	if len(buffer) < 4 {
		logs.Log.WithField("The lenght of the buffer is too small", len(buffer)).Error("Header size is not enough")
		return &Message{}, errors.New("size of message too small")
	}
	err := json.Unmarshal(buffer, &msg)
	if err != nil {
		logs.Log.WithField("error unmarshaling", err).Error("An error ocurred while reading message")
		return &Message{}, fmt.Errorf("error unmarshaling: %v", err)
	}
	return &msg, nil
}

func RecieveMessage(conn net.Conn) ([]byte, error) {
	logs.Initialize()
	header := make([]byte, 4)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		logs.Log.WithField("invalid header", err).Error("Error reading header ")
		return nil, fmt.Errorf("error reading header :%v", err)
	}
	messageLenght := binary.BigEndian.Uint32(header)
	message := make([]byte, messageLenght)
	_, err = io.ReadFull(conn, message)
	if err != nil {
		logs.Log.WithField("error reading message", err).Error("could recieve the message")
		return nil, fmt.Errorf("error reading message: %v", err)
	}
	logs.Log.WithField("Message lenght received was", messageLenght).Info("message received")
	return message, nil
}
