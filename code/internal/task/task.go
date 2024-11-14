package task

type Task struct {
	id          int
	idDAGNode   string
	chunk       []float64
	lastVersion int
	lastResult  float64
	isFinished  bool
}

func NewTask(id int, idDAGNode string, chunk []float64) *Task {
	return &Task{
		id:        id,
		idDAGNode: idDAGNode,
		chunk:     chunk,
	}
}

func (t *Task) Id() int {
	return t.id
}

func (t *Task) IdDAGNode() string {
	return t.idDAGNode
}

func (t *Task) Chunk() []float64 {
	return t.chunk
}

func (t *Task) LastVersion() int {
	return t.lastVersion
}

func (t *Task) LastResult() float64 {
	return t.lastResult
}
func (t *Task) SetId(id int) {
	t.id = id
}

func (t *Task) SetIDdagNode(idDAGNode string) {
	t.idDAGNode = idDAGNode
}

func (t *Task) SetChunk(chunk []float64) {
	t.chunk = chunk
}

func (t *Task) SetLastVersion(lastVersion int) {
	t.lastVersion = lastVersion
}

func (t *Task) SetLastResult(lastResult float64) {
	t.lastResult = lastResult
}

func (t *Task) SetFinished() {
	t.isFinished = true
}

func SafeTaskAccess(slice []Task, index int) (Task, bool) {
	if index >= 0 && index < len(slice) {
		return slice[index], true
	}
	return Task{}, false
}
