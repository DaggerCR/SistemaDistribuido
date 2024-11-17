package process

import (
	"sync"

	"distributed-system/internal/task"
	"distributed-system/pkg/utils"
)

type Process struct {
	id           utils.ProcId
	res          float64
	mu           sync.Mutex
	dependencies map[utils.TaskId]task.Task //maps task to its id
}

func NewProcess(id utils.ProcId) *Process {
	return &Process{
		id:           id,
		res:          0,
		dependencies: make(map[utils.TaskId]task.Task),
	}
}

func (p *Process) Id() utils.ProcId {
	return p.id
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

func (p *Process) AppendDependencies(dependency task.Task) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dependencies[dependency.Id] = dependency
}

func (p *Process) AugmentRes(val float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.res += val
}
