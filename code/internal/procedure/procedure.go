package procedure

import (
	"sync"

	"github.com/heimdalr/dag"
)

type Procedure struct {
	id           int
	res          float64
	mu           sync.Mutex
	dependencies map[string]bool //maps DAG node names to the finish status of task
	DAG          *dag.DAG
}

func (p *Procedure) SetRes(val float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.res = val
}

func (p *Procedure) Res() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.res
}

func (p *Procedure) SetDAG(DAG *dag.DAG) {
	p.DAG = DAG
}
