package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/sony/gobreaker"
)

// Job represents a job with dependencies and a payload
type Job struct {
	ID           string   `db:"id"`
	Payload      string   `db:"payload"`
	Dependencies []string `db:"dependencies"`
	MaxRetries   int      `db:"max_retries"`
	RetryCount   int      `db:"retry_count"`
	Priority     int      `db:"priority"`
	Status       string   `db:"status"`
}

// JobQueue manages job scheduling and execution
type JobQueue struct {
	mu       sync.Mutex
	queue    []*Job
	db       *sqlx.DB
	redis    *redis.Client
	cb       *gobreaker.CircuitBreaker
	depGraph map[string][]string // DAG for dependencies
}

// NewJobQueue creates a new JobQueue
func NewJobQueue(db *sqlx.DB, redis *redis.Client, cb *gobreaker.CircuitBreaker) *JobQueue {
	return &JobQueue{
		queue:    []*Job{},
		db:       db,
		redis:    redis,
		cb:       cb,
		depGraph: make(map[string][]string),
	}
}

// AddJob adds a job to the queue
func (jq *JobQueue) AddJob(job *Job) error {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	// Check if dependencies are satisfied
	for _, dep := range job.Dependencies {
		if jq.depGraph[dep] == nil {
			return fmt.Errorf("unsatisfied dependency: %s", dep)
		}
	}

	jq.queue = append(jq.queue, job)
	return nil
}

// ExecuteJobs processes jobs from the queue
func (jq *JobQueue) ExecuteJobs(ctx context.Context) {
	for {
		jq.mu.Lock()
		if len(jq.queue) == 0 {
			jq.mu.Unlock()
			time.Sleep(1 * time.Second)
			continue
		}

		job := jq.queue[0]
		jq.queue = jq.queue[1:]
		jq.mu.Unlock()

		// Execute the job
		go jq.executeJob(ctx, job)
	}
}

func (jq *JobQueue) executeJob(ctx context.Context, job *Job) {
	// Acquire distributed lock
	lockKey := fmt.Sprintf("job-lock-%s", job.ID)
	lockAcquired, err := jq.redis.SetNX(ctx, lockKey, 1, 5*time.Minute).Result()
	if err != nil || !lockAcquired {
		log.Printf("Failed to acquire lock for job %s", job.ID)
		return
	}
	defer jq.redis.Del(ctx, lockKey)

	// Circuit breaker execution
	_, err = jq.cb.Execute(func() (interface{}, error) {
		log.Printf("Executing job %s", job.ID)

		// Simulate job processing
		time.Sleep(time.Duration(rand.Intn(3)) * time.Second)

		// Simulate job failure
		if rand.Intn(2) == 0 && job.RetryCount < job.MaxRetries {
			job.RetryCount++
			backoffDuration := time.Duration(rand.Intn(3)) * time.Second
			log.Printf("Job %s failed, retrying in %v (Retry %d/%d)", job.ID, backoffDuration, job.RetryCount, job.MaxRetries)
			time.Sleep(backoffDuration)
			jq.AddJob(job) // Re-add job to queue
			return nil, fmt.Errorf("job failed")
		}

		if job.RetryCount >= job.MaxRetries {
			job.Status = "failed"
			log.Printf("Job %s failed after max retries", job.ID)
		} else {
			job.Status = "completed"
			log.Printf("Job %s completed successfully", job.ID)
		}

		// Update job status in the database
		jq.updateJobStatus(job)
		return nil, nil
	})

	if err != nil {
		log.Printf("Circuit breaker triggered for job %s: %v", job.ID, err)
	}
}

func (jq *JobQueue) updateJobStatus(job *Job) {
	_, err := jq.db.Exec("UPDATE jobs SET status=$1, retry_count=$2 WHERE id=$3", job.Status, job.RetryCount, job.ID)
	if err != nil {
		log.Printf("Error updating job status: %v", err)
	}
}

// GenerateReport generates a simple report of job execution statistics
func (jq *JobQueue) GenerateReport() {
	jobs := []*Job{}
	err := jq.db.Select(&jobs, "SELECT * FROM jobs ORDER BY priority DESC")
	if err != nil {
		log.Printf("Error generating report: %v", err)
		return
	}

	fmt.Println("Job Execution Report:")
	for _, job := range jobs {
		fmt.Printf("Job ID: %s, Status: %s, Retries: %d\n", job.ID, job.Status, job.RetryCount)
	}
}
