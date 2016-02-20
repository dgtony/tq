// Package tq implements simple lightweight task queue with pool of general workers.
// All tasks should be emitted from their origin goroutines to common task queue, and
// after that results will be received on individual response channels, contained in tasks.
// Package could be used in REST APIs i.e. to enqueue heavy tasks from http handlers.
package tq

import (
	"math/rand"
)

type TaskRequester interface {
	Compute()
}

// General task
type Task struct {
	TaskID  int
	ResChan chan<- Task
	Data    TaskRequester
}

// Run task queue and get channel for incoming tasks
func TaskQueue(numWorkers int) chan Task {
	processQueue := make(chan Task)
	go RunQueue(numWorkers, processQueue)
	return processQueue
}

// Get task and channel to listen on for the results
func CreateTask(request TaskRequester) (Task, chan Task) {
	rCh := make(chan Task)
	task := Task{
		TaskID:  rand.Int(),
		ResChan: rCh,
		Data:    request,
	}
	return task, rCh
}

func Worker(id int, in <-chan Task, res chan<- Task) {
	for t := range in {
		t.Data.Compute()
		res <- t
	}
}

func RunQueue(workersNum int, req <-chan Task) {
	commPool := make(map[int]chan<- Task)
	// initialize workers
	workerTaskQueue := make(chan Task, workersNum)
	workerTaskResults := make(chan Task, workersNum)
	defer func() {
		close(workerTaskQueue)
		close(workerTaskResults)
	}()
	for i := 0; i < workersNum; i++ {
		go Worker(i, workerTaskQueue, workerTaskResults)
	}
	// Main operation loop
	for {
		select {
		// dispatch incoming tasks
		case newTask := <-req:
			// add response channel to active pool
			commPool[newTask.TaskID] = newTask.ResChan
			// send task to workers in non-blocking way
			go func() {
				workerTaskQueue <- newTask
			}()
		// dispatch results provided by workers
		case newResult := <-workerTaskResults:
			// send result back to handler
			commPool[newResult.TaskID] <- newResult
			// delete from active pool
			delete(commPool, newResult.TaskID)
		}
	}
}
