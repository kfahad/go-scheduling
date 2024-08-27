package main

import (
	"context"
	"time"
)

// JobDependencyResolved checks if a job's dependencies are resolved
func (jq *JobQueue) JobDependencyResolved(job *Job) bool {
	for _, dep := range job.Dependencies {
		var status string
		err := jq.db.Get(&status, "SELECT status FROM jobs WHERE id=$1", dep)
		if err != nil {
			return false
		}
		if status != "completed" {
			return false
		}
	}
	return true
}

// ResolveDependencies resolves job dependencies and adds ready jobs to the queue
func (jq *JobQueue) ResolveDependencies(ctx context.Context) {
	for {
		jq.mu.Lock()
		for _, job := range jq.queue {
			if jq.JobDependencyResolved(job) {
				go jq.executeJob(ctx, job)
			}
		}
		jq.mu.Unlock()
		time.Sleep(2 * time.Second)
	}
}
