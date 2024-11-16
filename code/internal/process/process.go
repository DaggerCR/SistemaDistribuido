package process

import (
	"sync"

	"distributed-system/internal/task"
)

type Process struct {
	id           int
	res          float64
	mu           sync.Mutex
	dependencies []task.Task //maps DAG node names to the finish status of task
}

func NewProcess(id int) *Process {
	return &Process{
		id:           id,
		res:          0,
		dependencies: []task.Task{},
	}
}

func (p *Process) SetRes(val float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.res = val
}

func (p *Process) Res() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.res
}

func (p *Process) AppendDependencies(dependancy task.Task) {
	p.dependencies = append(p.dependencies, dependancy)
}
