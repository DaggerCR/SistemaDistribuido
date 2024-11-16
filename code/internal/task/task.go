package task

type Task struct {
	Id         int `json:"IdTask"`
	IdProc     int
	Chunk      []float64
	IsFinished bool
}

func NewTask(id int, idProc int, chunk []float64) *Task {
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
