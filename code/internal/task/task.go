package task

import (
	"distributed-system/pkg/utils"
	"fmt"
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
	return fmt.Sprintf("task id: %v,\t process id:%v,\t chunk: %v", t.Id, t.IdProc, t.Chunk)
}
