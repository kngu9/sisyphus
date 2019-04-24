// Copyright 2019 CanonicalLtd

package simulation

import (
	"context"
	"time"
)

type job func()

// newExecutor creates an executor that will run the
// specified number of workers to process jobs from the
// queue.
func newExecutor(numberOfWorkers int) *executor {
	e := &executor{
		queue: make(chan job, 16*numberOfWorkers),
	}
	for i := 0; i < numberOfWorkers; i++ {
		d := i
		go func() {
			e.worker(d)
		}()
	}
	return e
}

type executor struct {
	queue chan job
}

// close releases the used resources.
func (e *executor) close() {
	close(e.queue)
}

// addJob will create a new timer that will fire after the delay period. Once
// the timer fires, the job is put on the execution queue.
func (e *executor) addJob(ctx context.Context, delay time.Duration, j job) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			e.queue <- j
		}
	}()
}

// worker consumes jobs from the queue and executes them.
func (e *executor) worker(i int) {
	for {
		job := <-e.queue
		if job == nil {
			// the queue channel has been closed - exit
			return
		} else {
			job()
		}
	}
}
