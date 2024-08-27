package main

import (
	"context"
	"log"
)

func main() {
	ctx := context.Background()

	// Set up database
	db, err := setupDatabase()
	if err != nil {
		log.Fatalf("Error setting up database: %v", err)
	}

	// Set up Redis
	redisClient := setupRedis()

	// Set up Circuit Breaker
	cb := setupCircuitBreaker()

	// Create job queue
	jobQueue := NewJobQueue(db, redisClient, cb)

	// Sample jobs with dependencies
	job1 := &Job{ID: "job1", Payload: "Job 1 payload", MaxRetries: 3, Status: "pending"}
	job2 := &Job{ID: "job2", Payload: "Job 2 payload", MaxRetries: 3, Status: "pending"}

	// Add jobs to the queue
	err = jobQueue.AddJob(job1)
	if err != nil {
		log.Fatalf("Failed to add job: %v", err)
	}

	err = jobQueue.AddJob(job2)
	if err != nil {
		log.Fatalf("Failed to add job: %v", err)
	}

	// Start job execution
	go jobQueue.ExecuteJobs(ctx)

	// Start dependency resolution
	go jobQueue.ResolveDependencies(ctx)

	// Block main routine to keep the application running
	select {}
}
