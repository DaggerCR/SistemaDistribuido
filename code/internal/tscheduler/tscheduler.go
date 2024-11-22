package tscheduler

import (
	"distributed-system/internal/message"
	"distributed-system/internal/process"
	"distributed-system/logs"
	"errors"
	"sync"

	"distributed-system/internal/task"
	"distributed-system/pkg/utils"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
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
	logs.Initialize()
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
	logs.Initialize()
	if len(tasks) == 0 {
		return
	}

	logs.Log.WithFields(logrus.Fields{
		"taskLenght": len(tasks),
		"processID":  tasks[0].IdProc,
	}).Infof("[DEBUG] Starting assignment for %d tasks (Process ID: %v)\n", len(tasks), tasks[0].IdProc)

	maxNodeLoad := ts.maxNodeTaskLoad // Cached for performance

	// Step 1: Gather eligible nodes
	eligibleNodes := ts.getEligibleNodes(connections, maxNodeLoad)
	if len(eligibleNodes) == 0 {
		logs.Log.Warn("[WARNING] No eligible nodes available for task assignment.")
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
		logs.Log.WithField("Could not assign %d tasks. Adding to waitlist.\n", len(unassignedTasks)).Warn("[WARNING]")
		ts.addMultipleToTaskWaitlist(unassignedTasks)
	}
	logs.Log.Debug("[DEBUG] Task assignment process completed.")
}

func (ts *TScheduler) getEligibleNodes(connections map[utils.NodeId]net.Conn, maxNodeLoad int) map[utils.NodeId]net.Conn {
	logs.Initialize()
	eligibleNodes := make(map[utils.NodeId]net.Conn)
	var mu sync.Mutex
	wg := sync.WaitGroup{}

	for nodeId, conn := range connections {
		wg.Add(1)
		go func(nodeId utils.NodeId, conn net.Conn) {
			defer wg.Done()
			load, ok := ts.GetLoadBalance(nodeId)
			if !ok {
				logs.Log.WithField("nodeId", nodeId).Infof("[INFO] Node %v has no load data; skipping.\n", nodeId)
				return
			}
			if load >= utils.Load(maxNodeLoad) {
				logs.Log.WithFields(logrus.Fields{
					"nodeId":      nodeId,
					"currentLoad": load,
					"maxLoad":     maxNodeLoad,
				}).Debugf("[DEBUG] Node %v is overloaded (%v/%v); skipping.\n", nodeId, load, maxNodeLoad)
				return
			}
			mu.Lock()
			eligibleNodes[nodeId] = conn
			mu.Unlock()
		}(nodeId, conn)
	}
	wg.Wait()
	logs.Log.WithField("eligibleNodesCount", len(eligibleNodes)).Debugf("[DEBUG] Found %d eligible nodes for assignment.\n", len(eligibleNodes))
	return eligibleNodes
}

func (ts *TScheduler) distributeTasksEvenly(tasks []task.Task, nodeIds []utils.NodeId, eligibleNodes map[utils.NodeId]net.Conn, maxNodeLoad int) []task.Task {
	logs.Initialize()
	var unassignedTasks []task.Task
	nodeCount := len(nodeIds)

	for i, task := range tasks {
		nodeId := nodeIds[i%nodeCount] // Round-robin assignment
		conn := eligibleNodes[nodeId]

		load, ok := ts.GetLoadBalance(nodeId)
		if !ok || load >= utils.Load(maxNodeLoad) {
			logs.Log.WithFields(logrus.Fields{
				"nodeId":      nodeId,
				"currentLoad": load,
				"maxLoad":     maxNodeLoad,
				"taskId":      task.Id,
			}).Debugf("[DEBUG] Node %v is overloaded (%v/%v). Adding task %v to waitlist.\n", nodeId, load, maxNodeLoad, task.Id)
			unassignedTasks = append(unassignedTasks, task)
			continue
		}

		// Send the task to the node
		content := fmt.Sprintf("[DEBUG] Assigning task %v (Process: %v) to node %v.", task.Id, task.IdProc, nodeId)
		logs.Log.WithFields(logrus.Fields{
			"taskId":    task.Id,
			"processId": task.IdProc,
			"nodeId":    nodeId,
		}).Debugf("[DEBUG] Assigning task %v (Process: %v) to node %v.", task.Id, task.IdProc, nodeId)
		msg := message.NewMessage(message.AsignTask, content, task, 0)

		if err := message.SendMessage(msg, conn); err != nil {
			logs.Log.WithFields(logrus.Fields{
				"taskId": task.Id,
				"nodeId": nodeId,
				"error":  err,
			}).Errorf("[ERROR] Failed to send task %v to node %v: %v\n", task.Id, nodeId, err)
			unassignedTasks = append(unassignedTasks, task)
			continue
		}

		// Task successfully sent, update the load and registry
		ts.AppendToTaskRegistry(nodeId, task)
		ts.IncrementLoadBalance(nodeId)
		logs.Log.WithFields(logrus.Fields{
			"taskId":  task.Id,
			"nodeId":  nodeId,
			"newLoad": load + 1,
			"maxLoad": maxNodeLoad,
		}).Debugf("[DEBUG] Task %v assigned to node %v. Load: %v/%v.\n", task.Id, nodeId, load+1, maxNodeLoad)
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
	logs.Initialize()
	ts.mu.Lock()
	defer ts.mu.Unlock()
	for _, task := range entries {
		ts.taskWaitlist[task.Id] = task
	}
	logs.Log.WithField("waitlistCount", len(entries)).
		Debugf("[DEBUG] Added %d tasks to the waitlist.\n", len(entries))
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

func (ts *TScheduler) CreateNewProcess(entryArray []float64, numNodes int, connections map[utils.NodeId]net.Conn) (utils.ProcId, error) {
	logs.Initialize()
	newProcessId := ts.GetRandomIdNotInProcesses()
	ts.processMu.Lock()
	defer ts.processMu.Unlock()

	if len(ts.processes) >= ts.maxSystemProcesses {
		return -1, fmt.Errorf("system at max capacity. Try later")
	}

	ts.processes[newProcessId] = process.NewProcess(newProcessId)
	tasks, _, err := ts.CreateTasks(entryArray, numNodes, newProcessId)
	if err != nil {
		return -1, fmt.Errorf("error creating tasks for process %v: %v", newProcessId, err)
	}

	ts.processes[newProcessId].AppendDependencies(tasks...)
	ts.AsignTasks(tasks, connections)
	logs.Log.WithField("Process created with ID", newProcessId).Info("[INFO]")
	return newProcessId, nil
}

// returns true if process is finished
func (ts *TScheduler) UpdateProcessRes(idProc utils.ProcId, res float64) bool {
	ts.processMu.Lock()
	logs.Initialize()
	defer ts.processMu.Unlock()
	procToUpdate, ok := ts.processes[idProc]
	if !ok {
		logs.Log.WithField("processId", idProc).
			Warnf("[WARNING] Received update for non-existing procedure %v, canceling operation...\n", idProc)
		return false
	}
	procToUpdate.AugmentRes(res)

	logs.Log.WithFields(logrus.Fields{
		"processId":    idProc,
		"augmentation": res,
	}).Debugf("[DEBUG] Result for process with id: %v was augmented by: %v\n", idProc, res)
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
	ts.processMu.Lock()
	defer ts.processMu.Unlock()
	processModified, ok := ts.processes[procId]
	if ok {
		processModified.UpdateTaskStatus(taskId, true)
		ts.processes[procId] = processModified
		return true
	}
	return false
}

func (ts *TScheduler) HandleProcessUpdate(task task.Task, nodeId utils.NodeId, res float64) bool {
	logs.Initialize()
	if ts.UpdateProcessRes(task.IdProc, res) {
		if !ts.DeleteRedundacyTaskStatus(task.Id, nodeId) {
			logs.Log.WithField("nodeId", nodeId).Warn("[WARNING] Couldnt find task with nodeId.")
			return false
		}
		if !ts.UpdateProcessTaskStatus(task.Id, task.IdProc) {
			logs.Log.WithFields(logrus.Fields{
				"processId": task.IdProc,
				"taskId":    task.Id,
			}).Warnf("[WARNING] Couldnt find process %v, with task: %v", task.IdProc, task.Id)
			return false
		}
		ts.DecrementLoadBalance(nodeId)
		ts.processMu.Lock()
		process, ok := ts.processes[task.IdProc]
		if !ok {
			logs.Log.Error("Error: could not find process after all checks?")
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
	logs.Initialize()
	defer ts.mu.Unlock()

	// Check if the node exists in the load balance map
	currentLoad, ok := ts.loadBalance[nodeId]
	if ok {
		ts.loadBalance[nodeId]++
		logs.Log.WithFields(logrus.Fields{
			"nodeId":       nodeId,
			"previousLoad": currentLoad,
			"newLoad":      currentLoad + 1,
		}).Debugf("[DEBUG] Incremented load for node %v. Previous load: %v, New load: %v\n", nodeId, currentLoad, currentLoad+1)
	} else {
		logs.Log.WithField("nodeId", nodeId).
			Warnf("[WARNING] Tried to increment load for unknown node %v. Operation skipped.\n", nodeId)
	}
}

func (ts *TScheduler) DecrementLoadBalance(nodeId utils.NodeId) {
	ts.mu.Lock()
	logs.Initialize()
	defer ts.mu.Unlock()

	// Check if the node exists in the load balance map
	currentLoad, ok := ts.loadBalance[nodeId]
	if ok {
		if currentLoad > 0 {
			ts.loadBalance[nodeId]--
			logs.Log.WithFields(logrus.Fields{
				"nodeId":       nodeId,
				"previousLoad": currentLoad,
				"newLoad":      currentLoad - 1,
			}).Debugf("[DEBUG] Decremented load for node %v. Previous load: %v, New load: %v\n", nodeId, currentLoad, currentLoad-1)
		} else {
			logs.Log.WithField("nodeId", nodeId).
				Warnf("[WARNING] Tried to decrement load for node %v, but load is already 0. Operation skipped.\n", nodeId)
		}
	} else {
		logs.Log.WithField("nodeId", nodeId).
			Errorf("[ERROR] Tried to decrement load for unknown node %v. Operation skipped.\n", nodeId)
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
	ts.mu.Lock()
	logs.Initialize()
	defer ts.mu.Unlock()

	tasks, ok := ts.taskRegistry[nodeId]
	if !ok {
		logs.Log.WithField("nodeId", nodeId).
			Debugf("[DEBUG] No tasks to reassign for node %v.\n", nodeId)
		return
	}

	for _, task := range tasks {
		logs.Log.WithFields(logrus.Fields{
			"taskId":    task.Id,
			"processId": task.IdProc,
			"nodeId":    nodeId,
		}).Warnf("[WARNING] Reassigning task %v (Process: %v) due to node %v failure.\n", task.Id, task.IdProc, nodeId)
		ts.taskWaitlist[task.Id] = task
	}

	// Safely remove node and decrement load only if valid
	_, exists := ts.loadBalance[nodeId]
	if exists {
		ts.loadBalance[nodeId] = 0 // Reset load to avoid inconsistencies
		logs.Log.WithField("nodeId", nodeId).
			Debugf("[DEBUG] Reset load balance for failed node %v.\n", nodeId)
	}

	delete(ts.taskRegistry, nodeId)
	delete(ts.loadBalance, nodeId)
}
