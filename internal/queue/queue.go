package queue

import (
	"context"
	"errors"
	"time"
)

// Job status constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

var (
	// ErrNoJobs is returned when no jobs are available
	ErrNoJobs = errors.New("no jobs available")
	// ErrJobNotFound is returned when a job is not found
	ErrJobNotFound = errors.New("job not found")
)

// Job represents a conversion job
type Job struct {
	ID          string        `json:"id"`
	InputPath   string        `json:"input_path"`
	OutputPath  string        `json:"output_path"`
	Status      string        `json:"status"`
	Priority    int           `json:"priority"`
	CreatedAt   time.Time     `json:"created_at"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Error       string        `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Queue interface for job queue operations
type Queue interface {
	// Enqueue adds a job to the queue
	Enqueue(ctx context.Context, job *Job) error

	// Dequeue retrieves and removes the next job from the queue
	Dequeue(ctx context.Context) (*Job, error)

	// GetJob retrieves a job by ID without removing it
	GetJob(ctx context.Context, id string) (*Job, error)

	// UpdateJob updates an existing job
	UpdateJob(job *Job) error

	// CancelJob cancels a pending job
	CancelJob(ctx context.Context, id string) error

	// ListJobs lists all jobs with optional filtering
	ListJobs(ctx context.Context, status string, limit int) ([]*Job, error)

	// Size returns the number of pending jobs
	Size() (int, error)

	// Clear removes all jobs from the queue
	Clear() error
}