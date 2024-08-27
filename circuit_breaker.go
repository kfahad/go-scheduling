package main

import (
	"time"

	"github.com/sony/gobreaker"
)

func setupCircuitBreaker() *gobreaker.CircuitBreaker {
	st := gobreaker.Settings{
		Name:        "JobCircuitBreaker",
		MaxRequests: 1,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
	}
	return gobreaker.NewCircuitBreaker(st)
}
