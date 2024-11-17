package task

import "distributed-system/pkg/utils"

type Task struct {
	Id         utils.TaskId `json:"IdTask"`
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

func SafeTaskAccess(slice []Task, index int) (Task, bool) {
	if index >= 0 && index < len(slice) {
		return slice[index], true
	}
	return Task{}, false
}
