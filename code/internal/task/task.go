package task

import (
	"distributed-system/logs"
	"distributed-system/pkg/utils"
	"fmt"

	"github.com/sirupsen/logrus"
)

type Task struct {
	Id         utils.TaskId
	IdProc     utils.ProcId
	Chunk      []float64
	IsFinished bool
}

func NewTask(id utils.TaskId, idProc utils.ProcId, chunk []float64) *Task {
	return &Task{
		Id:         id,
		IdProc:     idProc,
		Chunk:      chunk,
		IsFinished: false,
	}
}

func (t *Task) Print() string {
	logs.Initialize()
	logs.Log.WithFields(logrus.Fields{
		"taskID":    t.Id,
		"processID": t.IdProc,
		"chunk":     t.Chunk,
	}).Infof("[INFO] Task id: %v,\t process id:%v,\t chunk: %v", t.Id, t.IdProc, t.Chunk)
	return fmt.Sprintf("task id: %v,\t process id:%v,\t chunk: %v", t.Id, t.IdProc, t.Chunk)
}
