package system

import (
	"distributed-system/internal/node"
	"distributed-system/internal/procedure"
)

type NodeId int
type Load int
type AccumulatedChecks int

type System struct {
	procedures     []*procedure.Procedure
	loadBalance    map[NodeId]Load //maps node id to its load
	systemNodes    []node.Node
	healthRegistry map[NodeId]AccumulatedChecks //maps node id to the amount of heart-beat checks since last heart-beat from node
}

func NewSystem() *System {
	return &System{}
}

// Procedures returns the slice of procedures in the system.
func (s *System) Procedures() []*procedure.Procedure {
	return s.procedures
}

// LoadBalance returns the load balance map in the system.
func (s *System) LoadBalance() map[NodeId]Load {
	return s.loadBalance
}

// SystemNodes returns the slice of nodes in the system.
func (s *System) SystemNodes() []node.Node {
	return s.systemNodes
}

// HealthRegistry returns the health registry map in the system.
func (s *System) HealthRegistry() map[NodeId]AccumulatedChecks {
	return s.healthRegistry
}

// Setters

// SetProcedures updates the procedures slice in the system.
func (s *System) SetProcedures(procedures []*procedure.Procedure) {
	s.procedures = procedures
}

// SetLoadBalance updates the load balance map in the system.
func (s *System) SetLoadBalance(loadBalance map[NodeId]Load) {
	s.loadBalance = loadBalance
}

// SetSystemNodes updates the system nodes slice in the system.
func (s *System) SetSystemNodes(systemNodes []node.Node) {
	s.systemNodes = systemNodes
}

// SetHealthRegistry updates the health registry map in the system.
func (s *System) SetHealthRegistry(healthRegistry map[NodeId]AccumulatedChecks) {
	s.healthRegistry = healthRegistry
}

// AppendProcedure adds a single procedure to the procedures slice.
func (s *System) AppendProcedure(proc *procedure.Procedure) {
	s.procedures = append(s.procedures, proc)
}

// AppendNode adds a single node to the systemNodes slice.
func (s *System) AppendNode(node node.Node) {
	s.systemNodes = append(s.systemNodes, node)
}

// Map Update Methods

// UpdateLoadBalance sets the load for a given NodeId in the loadBalance map.
func (s *System) UpdateLoadBalance(nodeID NodeId, load Load) {
	if s.loadBalance == nil {
		s.loadBalance = make(map[NodeId]Load)
	}
	s.loadBalance[nodeID] = load
}

// UpdateHealthRegistry updates the health registry for a given NodeId.
func (s *System) UpdateHealthRegistry(nodeID NodeId, check AccumulatedChecks) {
	if s.healthRegistry == nil {
		s.healthRegistry = make(map[NodeId]AccumulatedChecks)
	}
	s.healthRegistry[nodeID] = check
}

func (s *System) RemoveLoadBalance(nodeID NodeId) {
	delete(s.loadBalance, nodeID)
}

func (s *System) RemoveHealthRegistry(nodeID NodeId) {
	delete(s.healthRegistry, nodeID)
}
