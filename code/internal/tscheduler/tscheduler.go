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
	taskRegistry       map[utils.NodeId][]task.Task //data redundancy (to manage task reasignment )
	taskWaitlist       map[utils.TaskId]task.Task   //queue for unasigned task (aka task that weren't able to fit due to load balancing or max load of all nodes reached)
	processes          map[utils.ProcId]*process.Process
	loadBalance        map[utils.NodeId]utils.Load // Regular map for load balancing
	maxNodeTaskLoad    int
	maxSystemProcesses int
	processMu          sync.Mutex   // Protects processes slice
	mu                 sync.RWMutex // Protects loadBalance, systemNodes, and healthRegistry
}

const DefaultMaxTasksPerNode = 3
const DefaultMaxProcesses = 9

// NewTScheduler initializes and returns a new TScheduler instance.
func NewTScheduler() *TScheduler {
	maxNodeTaskLoadEnv := DefaultMaxTasksPerNode
	if envValLoad, err := strconv.Atoi(os.Getenv("MAX_TASKS")); err == nil {
		maxNodeTaskLoadEnv = envValLoad
	}
	maxSystemProcessesEnv := DefaultMaxProcesses
	if envValTask, err := strconv.Atoi(os.Getenv("MAX_PROCESSES")); err == nil {
		maxSystemProcessesEnv = envValTask
	}
	return &TScheduler{
		loadBalance:        make(map[utils.NodeId]utils.Load),
		taskRegistry:       make(map[utils.NodeId][]task.Task),
		taskWaitlist:       make(map[utils.TaskId]task.Task),
		processes:          make(map[utils.ProcId]*process.Process),
		maxNodeTaskLoad:    maxNodeTaskLoadEnv,
		maxSystemProcesses: maxSystemProcessesEnv,
	}
}

// Getters

// TaskRegistry returns the slice of tasks in the taskRegistry.
func (ts *TScheduler) TaskRegistry() map[utils.NodeId][]task.Task {
	return ts.taskRegistry
}

// TaskQueue returns the slice of tasks in the taskWaitlist.
func (ts *TScheduler) TaskWaitlist() map[utils.TaskId]task.Task {
	return ts.taskWaitlist
}

// Setters

// SetTaskQueue updates the taskWaitlist slice.
func (ts *TScheduler) SetTaskWaitlist(tasks map[utils.TaskId]task.Task) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.taskWaitlist = tasks
}

// Append Methods

// AppendToTaskRegistry adds a single task to the taskRegistry slice.
func (ts *TScheduler) AppendToTaskRegistry(nodeId utils.NodeId, task task.Task) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.taskRegistry[nodeId] = append(ts.taskRegistry[nodeId], task)
}

// AppendToTaskQueue adds a single task to the taskWaitlist slice.
func (ts *TScheduler) AppendToTaskWaitlist(taskId utils.TaskId, task task.Task) {
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
	fmt.Println("LOAD FOR THIS MF IS: ", load)
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

func (ts *TScheduler) GetTaskFromWaitlist(taskId utils.TaskId) (task.Task, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	task, exists := ts.taskWaitlist[taskId]
	return task, exists
}

func (ts *TScheduler) RemoveFromTaskWaitlist(taskId utils.TaskId) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.taskWaitlist, taskId)
}

func (ts *TScheduler) GetProcess(procId utils.ProcId) (*process.Process, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	process, exists := ts.processes[procId]
	return process, exists
}

func (ts *TScheduler) RemoveProcess(procId utils.ProcId) {
	ts.processMu.Lock()
	defer ts.processMu.Unlock()
	delete(ts.processes, procId)
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

	for nodeId, load := range ts.loadBalance {
		if load <= utils.Load(ts.maxNodeTaskLoad) {
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
		taskId := int(idProc)*1000 + idx
		newTask := task.NewTask(utils.TaskId(taskId), idProc, chunk)
		tasks = append(tasks, *newTask)
	}
	return tasks, len(tasks), nil
}

func (ts *TScheduler) AsignTasks(tasks []task.Task, connections map[utils.NodeId]net.Conn) {
	idx := 0 // Tracks the number of tasks assigned so far

	// Retrieve the maximum allowed load per node from the environment or use a default
	maxNodeLoad := ts.maxNodeTaskLoad // Use cached maxNodeLoad
	// Step 1: Single pass to assign tasks and collect underloaded nodes
	underloadedNodes := []utils.NodeId{} // Nodes with capacity for more tasks
	for nodeId, conn := range connections {
		if idx >= len(tasks) {
			break // All tasks have been assigned
		}

		// Check the current load of the node
		load, found := ts.GetLoadBalance(nodeId)
		if !found || load >= utils.Load(maxNodeLoad) {
			fmt.Printf("\n Skipping over because found : %v or overload %v/%v \n", found, load, maxNodeLoad)
			continue // Skip overloaded or unknown nodes
		}

		// Assign the task to this node
		task := tasks[idx]
		content := fmt.Sprintf("\nAssigned task with id %v from process: %v to node: %v\n", task.Id, task.IdProc, nodeId)
		msg := message.NewMessage(message.AsignTask, content, task, 0)
		if err := message.SendMessage(msg, conn); err != nil {
			fmt.Printf("\nError assigning task %v to node %v: %v; assigned to waitlist\n", task.Id, nodeId, err)
			ts.AppendToTaskWaitlist(task.Id, task)
			continue
		}
		ts.AppendToTaskRegistry(nodeId, task)
		idx++ // Successfully assigned this task
		// Track this node as underloaded for potential reassignment
		if load+1 < utils.Load(maxNodeLoad) {
			underloadedNodes = append(underloadedNodes, nodeId)
		}
	}
	// Step 2: Assign remaining tasks to underloaded nodes
	for _, nodeId := range underloadedNodes {
		if idx >= len(tasks) {
			break // All tasks have been assigned
		}

		// Assign remaining tasks
		task := tasks[idx]
		conn := connections[nodeId]
		content := fmt.Sprintf("Assigned task with id %v from process: %v to node: %v", task.Id, task.IdProc, nodeId)
		msg := message.NewMessage(message.AsignTask, content, task, 0)
		if err := message.SendMessage(msg, conn); err != nil {
			fmt.Printf("Error assigning task %v to node %v: %v; assigned to waitlist\n", task.Id, nodeId, err)
			ts.AppendToTaskWaitlist(task.Id, task)
			continue
		}

		ts.AppendToTaskRegistry(nodeId, task)
		idx++
	}

	// Step 3: Handle tasks that could not be assigned
	if idx < len(tasks) {
		ts.addMultipleToTaskWaitlist(tasks[idx:])
		fmt.Printf("Could not assign %d tasks, added to waitlist\n", len(tasks)-idx)
	}
}

func (ts *TScheduler) AttemptReasign(connections map[utils.NodeId]net.Conn) {
	ts.mu.Lock()
	currentTasksWaitlist := ts.TaskWaitlist()
	var currentUnasignedTasks []task.Task
	for _, task := range currentTasksWaitlist {
		currentUnasignedTasks = append(currentUnasignedTasks, task)
	}
	ts.mu.Unlock()
	ts.SetTaskWaitlist(make(map[utils.TaskId]task.Task))
	ts.AsignTasks(currentUnasignedTasks, connections)
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
	if len(ts.processes) >= ts.maxNodeTaskLoad {
		return 0, errors.New("system is at maximum capacity, try later")
	}
	ts.processes[newProcessId] = process.NewProcess(newProcessId)
	tasks, numTasks, err := ts.CreateTasks(entryArray, numNodes, newProcessId)
	if err != nil {
		initialErrorMsg := fmt.Sprintf("error creating tasks for process with id: %v ", newProcessId)
		return 0, fmt.Errorf(initialErrorMsg+": %v", err)
	}
	ts.processes[newProcessId].AppendDependencies(tasks...)
	ts.AsignTasks(tasks, connections)

	fmt.Printf("\nProcess created with id %v", newProcessId)

	return numTasks, nil
}

// returns true if process is finished
func (ts *TScheduler) UpdateProcessRes(idProc utils.ProcId, res float64) bool {
	procToUpdate, ok := ts.GetProcess(idProc)
	if !ok {
		fmt.Printf("\nRecieved update for non existing procedure %v, canceling operation...\n", idProc)
		return false
	}
	procToUpdate.AugmentRes(res)
	fmt.Printf("\nResult for process with id: %v was augmneted by :%v \n", idProc, res)
	return true
}

func (ts *TScheduler) UpdateRedundacyTaskStatus(taskId utils.TaskId, nodeId utils.NodeId) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	tasks, ok := ts.taskRegistry[nodeId]
	if ok {
		for idx, task := range tasks {
			if task.Id == taskId {
				task.IsFinished = true
				ts.taskRegistry[nodeId][idx] = task
				break
			}
		}

		return true
	}
	return false
}

func (ts *TScheduler) UpdateProcessTaskStatus(taskId utils.TaskId, procId utils.ProcId) bool {
	//fmt.Println("TEST2.X")
	ts.processMu.Lock()
	defer ts.processMu.Unlock()
	//fmt.Println("TEST2.2")
	processModified, ok := ts.processes[procId]
	if ok {
		//fmt.Println("TEST2.3")
		processModified.UpdateTaskStatus(taskId, true)
		//fmt.Println("TEST2.4")
		ts.processes[procId] = processModified
		return true
	}
	return false
}

func (ts *TScheduler) HandleProcessUpdate(task task.Task, nodeId utils.NodeId, res float64) bool {
	if ts.UpdateProcessRes(task.IdProc, res) {
		//fmt.Println("TEST1")
		if !ts.DeleteRedundacyTaskStatus(task.Id, nodeId) {
			fmt.Println("Couldnt find task with nodeId: ", nodeId)
			return false
		}
		//fmt.Println("TEST2")
		if !ts.UpdateProcessTaskStatus(task.Id, task.IdProc) {
			fmt.Println("Couldnt find process with task: ", task.Id)
			return false
		}
		//fmt.Println("TEST2.5")
		currentLoad, ok := ts.GetLoadBalance(nodeId)
		//fmt.Println("TEST3")
		if !ok {
			fmt.Println("Corrupt node")
			return false
		}
		currentLoad--
		if currentLoad < 0 {
			fmt.Println("Error load is negative?")
		}
		//fmt.Println("TEST4")
		ts.UpdateLoadBalance(nodeId, currentLoad)
		process, ok := ts.GetProcess(task.IdProc)
		if !ok {
			fmt.Println("Error could not find process after all checks?")
		}
		//fmt.Println("TEST5")
		if process.CheckFinished() {
			ts.RemoveProcess(task.IdProc)
		}
		//fmt.Println("TEST6")
		return true
	}
	return false
}

func (ts *TScheduler) DeleteRedundacyTaskStatus(taskId utils.TaskId, nodeId utils.NodeId) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	tasksToCheck, ok := ts.taskRegistry[nodeId]
	if ok {
		for idx, task := range tasksToCheck {
			if task.Id == taskId {
				ts.taskRegistry[nodeId] = append(tasksToCheck[:idx], tasksToCheck[idx+1:]...)
				return true
			}
		}
	}
	return false
}
