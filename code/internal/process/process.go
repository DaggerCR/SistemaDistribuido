package process

import (
	"sync"

	"distributed-system/internal/task"
	"distributed-system/logs"
	"distributed-system/pkg/utils"

	"github.com/sirupsen/logrus"
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

func (p *Process) AppendDependencies(dependencies ...task.Task) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, dependency := range dependencies {
		p.dependencies[dependency.Id] = dependency
	}
}

func (p *Process) AugmentRes(val float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.res += val
}

func (p *Process) CheckFinished() bool {
	logs.Initialize()
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, dependency := range p.dependencies {
		if !dependency.IsFinished {
			return false
		}
	}
	logs.Log.WithFields(logrus.Fields{
		"ProcessID": p.id,
		"result":    p.res,
	}).Infof("[INFO]Process with id: %v has finished with final result: %v", p.id, p.res)

	return true
}

func (p *Process) UpdateTaskStatus(taskId utils.TaskId, status bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	taskModified, ok := p.dependencies[taskId]
	if ok {
		taskModified.IsFinished = status
		p.dependencies[taskId] = taskModified
	}
}

func (p *Process) Dependencies() map[utils.TaskId]task.Task {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.dependencies
}
