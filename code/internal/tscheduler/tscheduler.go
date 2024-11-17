package tscheduler

import (
	"distributed-system/internal/message"
	"distributed-system/internal/process"
	"errors"
	"sort"
	"sync"

	"distributed-system/internal/task"
	"distributed-system/pkg/utils"
	"fmt"
	"net"
	"os"
	"strconv"
)

type TScheduler struct {
	taskRegistry       map[utils.NodeId]task.Task //data redundancy (to manage task reasignment )
	taskWaitlist       map[utils.TaskId]task.Task //queue for unasigned task (aka task that weren't able to fit due to load balancing or max load of all nodes reached)
	processes          map[utils.ProcId]*process.Process
	loadBalance        map[utils.NodeId]utils.Load // Regular map for load balancing
	defaultMaxNodeLoad int
	processMu          sync.Mutex   // Protects processes slice
	mu                 sync.RWMutex // Protects loadBalance, systemNodes, and healthRegistry
}

// NewTScheduler initializes and returns a new TScheduler instance.
func NewTScheduler(defaultMaxNodeLoad int) *TScheduler {
	return &TScheduler{
		loadBalance:        make(map[utils.NodeId]utils.Load),
		taskRegistry:       make(map[utils.NodeId]task.Task),
		taskWaitlist:       make(map[utils.TaskId]task.Task),
		processes:          make(map[utils.ProcId]*process.Process),
		defaultMaxNodeLoad: defaultMaxNodeLoad,
	}
}

// Getters

// TaskRegistry returns the slice of tasks in the taskRegistry.
func (ts *TScheduler) TaskRegistry() map[utils.NodeId]task.Task {
	return ts.taskRegistry
}

// TaskQueue returns the slice of tasks in the taskWaitlist.
func (ts *TScheduler) TaskQueue() map[utils.TaskId]task.Task {
	return ts.taskWaitlist
}

// Setters

// SetTaskQueue updates the taskWaitlist slice.
func (ts *TScheduler) SetTaskQueue(tasks map[utils.TaskId]task.Task) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.taskWaitlist = tasks
}

// Append Methods

// AppendToTaskRegistry adds a single task to the taskRegistry slice.
func (ts *TScheduler) AppendToTaskRegistry(nodeId utils.NodeId, task task.Task) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.taskRegistry[nodeId] = task
}

// AppendToTaskQueue adds a single task to the taskWaitlist slice.
func (ts *TScheduler) AppendToTaskQueue(taskId utils.TaskId, task task.Task) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.taskWaitlist[taskId] = task
}

// UpdateLoadBalance sets the load for a given utils.NodeId.
func (ts *TScheduler) UpdateLoadBalance(nodeId utils.NodeId, load utils.Load) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.loadBalance[nodeId] = load
}

// GetLoadBalance retrieves the load for a given utils.NodeId.
func (ts *TScheduler) GetLoadBalance(nodeId utils.NodeId) (utils.Load, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	load, ok := ts.loadBalance[nodeId]
	return load, ok
}

// RemoveLoadBalance removes a node from the load balance map.
func (ts *TScheduler) RemoveFromLoadBalance(nodeId utils.NodeId) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.loadBalance, nodeId)
}

func (ts *TScheduler) AddToLoadBalance(nodeId utils.NodeId) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.loadBalance[nodeId] = 0
}

// Processes returns the slice of processes in the system.
func (ts *TScheduler) Processes() map[utils.ProcId]*process.Process {
	ts.processMu.Lock()
	defer ts.processMu.Unlock()
	return ts.processes
}

// SetProcesses updates the processes slice in the system.
func (ts *TScheduler) SetProcesses(processes map[utils.ProcId]*process.Process) {
	ts.processMu.Lock()
	defer ts.processMu.Unlock()
	ts.processes = processes
}

// ProbeLessLoadedNodes retrieves nodes with the least load while considering node connections.
func (ts *TScheduler) ProbeLessLoadedNodes(amount int, systemNodes map[utils.NodeId]net.Conn) []utils.NodeId {
	ts.mu.RLock() // Lock for loadBalance access
	defer ts.mu.RUnlock()

	type nodeLoad struct {
		nodeId utils.NodeId
		load   utils.Load
	}

	var nodes []nodeLoad
	maxNodeLoad, err := strconv.Atoi(os.Getenv("MAX_TASKS"))
	if err != nil {
		fmt.Printf("Invalid MAX_TASKS value: %v\n", err)
		maxNodeLoad = ts.defaultMaxNodeLoad
	}

	for nodeId, load := range ts.loadBalance {
		if load <= utils.Load(maxNodeLoad) {
			if _, exists := systemNodes[nodeId]; exists { // Ensure the node is still connected
				nodes = append(nodes, nodeLoad{nodeId, load})
			}
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].load < nodes[j].load
	})

	result := make([]utils.NodeId, 0, amount)
	for i := 0; i < len(nodes) && i < amount; i++ {
		result = append(result, nodes[i].nodeId)
	}
	return result
}

func (ts *TScheduler) CreateTasks(entryArray []float64, numNodes int, idProc utils.ProcId) ([]task.Task, int, error) {
	//numNodes never passes as less than or equal 0
	if numNodes <= 0 {
		return []task.Task{}, 0, errors.New("numNodes must be greater than 0")
	}
	if len(entryArray) == 0 {
		return []task.Task{}, 0, nil // No entries to process
	}
	chunkLen := (len(entryArray) + numNodes - 1) / numNodes
	var tasks []task.Task
	chunks, err := utils.SliceUpArray(entryArray, chunkLen)
	if err != nil {
		return nil, 0, fmt.Errorf("could not create task for procedure in this moment: %w", err)
	}
	for idx, chunk := range chunks {
		taskId := int(idProc) + idx
		newTask := task.NewTask(utils.TaskId(taskId), idProc, chunk)
		tasks = append(tasks, *newTask)
	}
	return tasks, len(tasks), nil
}

func (ts *TScheduler) asignTasks(tasks []task.Task, connections map[utils.NodeId]net.Conn) {
	idx := 0 // Tracks the number of tasks assigned so far

	// Retrieve the maximum allowed load per node from the environment or use a default
	maxNodeLoad, err := strconv.Atoi(os.Getenv("MAX_TASKS"))
	if err != nil {
		maxNodeLoad = ts.defaultMaxNodeLoad
	}

	// Step 1: Attempt to assign tasks to available nodes within their load capacity
	// Only nodes with load < maxNodeLoad are considered
	if len(tasks) >= len(connections) { // More tasks than nodes
		for nodeId, conn := range connections {
			if idx >= len(tasks) {
				break // All tasks have been assigned
			}

			// Check the current load of the node
			load, found := ts.GetLoadBalance(utils.NodeId(nodeId))
			if found && load >= utils.Load(maxNodeLoad) {
				continue // Skip nodes that are already at or above max load
			}

			// Create and send the task assignment message
			content := fmt.Sprintf("Assigned task with id %v from process: %v", tasks[idx].Id, tasks[idx].IdProc)
			msg := message.NewMessage(message.AsignTask, content, tasks[idx], 0)
			if err := message.SendMessage(*msg, conn); err != nil {
				// If sending fails, requeue the task for later assignment
				ts.AppendToTaskQueue(tasks[idx].Id, tasks[idx])
				continue
			}
			ts.AppendToTaskRegistry(nodeId, tasks[idx])
			idx++ // Successfully assigned this task
		}
	}

	// Step 2: Distribute remaining tasks among the least-loaded nodes
	// This block is entered only if there are pending tasks
	if idx < len(tasks) {
		// Identify nodes with the least load to handle remaining tasks
		lessLoadedNodes := ts.ProbeLessLoadedNodes(len(tasks)-idx, connections)
		for _, nodeId := range lessLoadedNodes {
			if idx >= len(tasks) {
				break // All tasks have been assigned
			}

			// Create and send the task assignment message
			content := fmt.Sprintf("Assigned task with id %v from process: %v", tasks[idx].Id, tasks[idx].IdProc)
			msg := message.NewMessage(message.AsignTask, content, tasks[idx], 0)
			if err := message.SendMessage(*msg, connections[utils.NodeId(nodeId)]); err != nil {
				// If sending fails, requeue the task for later assignment
				ts.AppendToTaskQueue(tasks[idx].Id, tasks[idx])
				continue
			}
			ts.AppendToTaskRegistry(nodeId, tasks[idx])
			idx++ // Successfully assigned this task
		}
	}

	// Step 3: Handle tasks that could not be assigned
	// Remaining tasks (if any) are appended to the task waitlist
	if idx < len(tasks) {
		ts.addMultipleToTaskWaitlist(tasks[idx:])
	}
}

func (ts *TScheduler) addMultipleToTaskWaitlist(entries []task.Task) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	for _, task := range entries {
		ts.taskWaitlist[task.Id] = task
	}
}

func (ts *TScheduler) GetRandomIdNotInProcesses() utils.ProcId {
	ts.processMu.Lock()
	defer ts.processMu.Unlock()

	// Create a set of process IDs for O(1) lookup
	idSet := make(map[utils.ProcId]struct{})
	for _, process := range ts.processes {
		idSet[process.Id()] = struct{}{}
	}

	// Generate a random ID that doesn't exist in the set
	for {
		num := int(utils.GenRandomUint())
		if _, exists := idSet[utils.ProcId(num)]; !exists {
			return utils.ProcId(num)
		}
	}
}

// MUST HANDLE ERROR ON CALLER
func (ts *TScheduler) CreateNewProcess(entryArray []float64, numNodes int, connections map[utils.NodeId]net.Conn) (int, error) {
	newProcessId := ts.GetRandomIdNotInProcesses()
	ts.processMu.Lock()
	defer ts.processMu.Unlock()
	ts.processes[newProcessId] = process.NewProcess(newProcessId)
	tasks, numTasks, err := ts.CreateTasks(entryArray, numNodes, newProcessId)
	if err != nil {
		initialErrorMsg := fmt.Sprintf("error creating tasks for process with id: %v ", newProcessId)
		return 0, fmt.Errorf(initialErrorMsg+": %v", err)
	}
	fmt.Printf("\n\n Task created were %v: \n %v \n\n", numTasks, tasks)
	ts.asignTasks(tasks, connections)
	return numTasks, nil
}

func (ts *TScheduler) UpdateProcessRes(idProc utils.ProcId, res float64) {
	ts.processMu.Lock()
	defer ts.processMu.Unlock()
	ts.processes[idProc].AugmentRes(res)
}
