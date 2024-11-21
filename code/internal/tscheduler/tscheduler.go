package tscheduler

import (
	"distributed-system/internal/message"
	"distributed-system/internal/process"
	"errors"
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
	//fmt.Printf("Load of node %d is: %d\n", nodeId, load)
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

func (ts *TScheduler) CreateTasks(entryArray []float64, numNodes int, idProc utils.ProcId) ([]task.Task, int, error) {
	//numNodes never passes as less than or equal 0
	if numNodes <= 0 {
		return []task.Task{}, 0, errors.New("numNodes must be greater than 0")
	}
	if len(entryArray) == 0 {
		return []task.Task{}, 0, nil // No entries to process
	}
	//chunkLen := (len(entryArray) + numNodes - 1) / numNodes
	chunkLen := 2
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
	if len(tasks) == 0 {
		return
	}

	fmt.Printf("[DEBUG] Starting assignment for %d tasks (Process ID: %v)\n", len(tasks), tasks[0].IdProc)

	maxNodeLoad := ts.maxNodeTaskLoad // Cached for performance

	// Step 1: Gather eligible nodes
	eligibleNodes := ts.getEligibleNodes(connections, maxNodeLoad)
	if len(eligibleNodes) == 0 {
		fmt.Println("[WARNING] No eligible nodes available for task assignment.")
		ts.addMultipleToTaskWaitlist(tasks)
		return
	}

	// Step 2: Distribute tasks evenly across eligible nodes
	nodeIds := make([]utils.NodeId, 0, len(eligibleNodes))
	for nodeId := range eligibleNodes {
		nodeIds = append(nodeIds, nodeId)
	}

	unassignedTasks := ts.distributeTasksEvenly(tasks, nodeIds, eligibleNodes, maxNodeLoad)

	// Step 3: Handle unassigned tasks
	if len(unassignedTasks) > 0 {
		fmt.Printf("[WARNING] Could not assign %d tasks. Adding to waitlist.\n", len(unassignedTasks))
		ts.addMultipleToTaskWaitlist(unassignedTasks)
	}

	fmt.Println("[DEBUG] Task assignment process completed.")
}

func (ts *TScheduler) getEligibleNodes(connections map[utils.NodeId]net.Conn, maxNodeLoad int) map[utils.NodeId]net.Conn {
	eligibleNodes := make(map[utils.NodeId]net.Conn)
	var mu sync.Mutex
	wg := sync.WaitGroup{}

	for nodeId, conn := range connections {
		wg.Add(1)
		go func(nodeId utils.NodeId, conn net.Conn) {
			defer wg.Done()
			load, ok := ts.GetLoadBalance(nodeId)
			if !ok {
				fmt.Printf("[INFO] Node %v has no load data; skipping.\n", nodeId)
				return
			}
			if load >= utils.Load(maxNodeLoad) {
				fmt.Printf("[DEBUG] Node %v is overloaded (%v/%v); skipping.\n", nodeId, load, maxNodeLoad)
				return
			}
			mu.Lock()
			eligibleNodes[nodeId] = conn
			mu.Unlock()
		}(nodeId, conn)
	}
	wg.Wait()
	fmt.Printf("[DEBUG] Found %d eligible nodes for assignment.\n", len(eligibleNodes))
	return eligibleNodes
}

func (ts *TScheduler) distributeTasksEvenly(tasks []task.Task, nodeIds []utils.NodeId, eligibleNodes map[utils.NodeId]net.Conn, maxNodeLoad int) []task.Task {
	var unassignedTasks []task.Task
	nodeCount := len(nodeIds)

	for i, task := range tasks {
		nodeId := nodeIds[i%nodeCount] // Round-robin assignment
		conn := eligibleNodes[nodeId]

		load, ok := ts.GetLoadBalance(nodeId)
		if !ok || load >= utils.Load(maxNodeLoad) {
			fmt.Printf("[DEBUG] Node %v is overloaded (%v/%v). Adding task %v to waitlist.\n", nodeId, load, maxNodeLoad, task.Id)
			unassignedTasks = append(unassignedTasks, task)
			continue
		}

		// Send the task to the node
		content := fmt.Sprintf("[DEBUG] Assigning task %v (Process: %v) to node %v.", task.Id, task.IdProc, nodeId)
		fmt.Println(content)
		msg := message.NewMessage(message.AsignTask, content, task, 0)

		if err := message.SendMessage(msg, conn); err != nil {
			fmt.Printf("[ERROR] Failed to send task %v to node %v: %v\n", task.Id, nodeId, err)
			unassignedTasks = append(unassignedTasks, task)
			continue
		}

		// Task successfully sent, update the load and registry
		ts.AppendToTaskRegistry(nodeId, task)
		ts.IncrementLoadBalance(nodeId)
		fmt.Printf("[DEBUG] Task %v assigned to node %v. Load: %v/%v.\n", task.Id, nodeId, load+1, maxNodeLoad)
	}

	return unassignedTasks
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
	fmt.Printf("[DEBUG] Added %d tasks to the waitlist.\n", len(entries))
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
func (ts *TScheduler) CreateNewProcess(entryArray []float64, numNodes int, connections map[utils.NodeId]net.Conn) {
	newProcessId := ts.GetRandomIdNotInProcesses()
	ts.processMu.Lock()
	defer ts.processMu.Unlock()
	if len(ts.processes) >= ts.maxSystemProcesses {
		fmt.Println("[WARNING] System is at maximum capacity, try later")
	}
	ts.processes[newProcessId] = process.NewProcess(newProcessId)
	tasks, _, err := ts.CreateTasks(entryArray, numNodes, newProcessId)
	if err != nil {
		println("[WARNING] Error: creating tasks for process with id: %v : %v", newProcessId, err)
	}
	ts.processes[newProcessId].AppendDependencies(tasks...)
	ts.AsignTasks(tasks, connections)
	fmt.Printf("[INFO] Process created with id %v\n", newProcessId)
}

// returns true if process is finished
func (ts *TScheduler) UpdateProcessRes(idProc utils.ProcId, res float64) bool {
	ts.processMu.Lock()
	defer ts.processMu.Unlock()
	procToUpdate, ok := ts.processes[idProc]
	if !ok {
		fmt.Printf("[WARNING] Recieved update for non existing procedure %v, canceling operation...\n", idProc)
		return false
	}
	procToUpdate.AugmentRes(res)

	fmt.Printf("[DEBUG] Result for process with id: %v was augmneted by :%v \n", idProc, res)
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
		if !ts.DeleteRedundacyTaskStatus(task.Id, nodeId) {
			fmt.Println("[WARNING] Couldnt find task with nodeId: ", nodeId)
			return false
		}
		if !ts.UpdateProcessTaskStatus(task.Id, task.IdProc) {
			fmt.Printf("[WARNING] Couldnt find process  %v, with task: %v \n", task.IdProc, task.Id)
			return false
		}
		ts.DecrementLoadBalance(nodeId)
		ts.processMu.Lock()
		process, ok := ts.processes[task.IdProc]
		if !ok {
			fmt.Println("Error could not find process after all checks?")
		}
		if process.CheckFinished() {
			delete(ts.processes, task.IdProc)
		}
		ts.processMu.Unlock()
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

func (ts *TScheduler) IncrementLoadBalance(nodeId utils.NodeId) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Check if the node exists in the load balance map
	currentLoad, ok := ts.loadBalance[nodeId]
	if ok {
		ts.loadBalance[nodeId]++
		fmt.Printf("[DEBUG] Incremented load for node %v. Previous load: %v, New load: %v\n", nodeId, currentLoad, currentLoad+1)
	} else {
		fmt.Printf("[WARNING] Tried to increment load for unknown node %v. Operation skipped.\n", nodeId)
	}
}

func (ts *TScheduler) DecrementLoadBalance(nodeId utils.NodeId) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Check if the node exists in the load balance map
	currentLoad, ok := ts.loadBalance[nodeId]
	if ok {
		if currentLoad > 0 {
			ts.loadBalance[nodeId]--
			fmt.Printf("[DEBUG] Decremented load for node %v. Previous load: %v, New load: %v\n", nodeId, currentLoad, currentLoad-1)
		} else {
			fmt.Printf("[WARNING] Tried to decrement load for node %v, but load is already 0. Operation skipped.\n", nodeId)
		}
	} else {
		fmt.Printf("[ERROR] Tried to decrement load for unknown node %v. Operation skipped.\n", nodeId)
	}
}

func (ts *TScheduler) GetTasksByNodeId(nodeId utils.NodeId) ([]task.Task, bool) {
	ts.mu.RLock() // Use read lock for safe concurrent access
	defer ts.mu.RUnlock()

	tasks, ok := ts.taskRegistry[nodeId]
	if !ok {
		return nil, false // Node ID not found in the registry
	}
	return tasks, true
}

func (ts *TScheduler) ReassignTasksForNode(nodeId utils.NodeId) {
	// Reassign tasks in taskRegistry for this node
	ts.mu.Lock()
	defer ts.mu.Unlock()
	tasks, ok := ts.taskRegistry[nodeId]
	if !ok {
		fmt.Printf("[DEBUG] No tasks to reassign for node %v.\n", nodeId)
		return
	}
	for _, task := range tasks {
		fmt.Printf("[WARNING] Reassigning task %v (Process: %v) due to node %v failure.\n", task.Id, task.IdProc, nodeId)
		ts.taskWaitlist[task.Id] = task
		ts.loadBalance[nodeId]--
	}
	// Remove the node from the registry
	delete(ts.taskRegistry, nodeId)
	delete(ts.loadBalance, nodeId)
}
