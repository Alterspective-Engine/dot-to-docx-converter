package queue

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MemoryQueue implements Queue interface using in-memory storage
type MemoryQueue struct {
	mu       sync.RWMutex
	jobs     map[string]*Job
	pending  []*Job
	nextPoll time.Time
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		jobs:    make(map[string]*Job),
		pending: make([]*Job, 0),
	}
}

// Enqueue adds a job to the queue
func (q *MemoryQueue) Enqueue(ctx context.Context, job *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Store job
	q.jobs[job.ID] = job

	// Add to pending queue if status is pending
	if job.Status == StatusPending || job.Status == "" {
		job.Status = StatusPending
		q.pending = append(q.pending, job)

		// Sort by priority (higher priority first) then by creation time
		sort.Slice(q.pending, func(i, j int) bool {
			if q.pending[i].Priority != q.pending[j].Priority {
				return q.pending[i].Priority > q.pending[j].Priority
			}
			return q.pending[i].CreatedAt.Before(q.pending[j].CreatedAt)
		})
	}

	return nil
}

// Dequeue retrieves the next job from the queue
func (q *MemoryQueue) Dequeue(ctx context.Context) (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if we should wait before polling
	if time.Now().Before(q.nextPoll) {
		return nil, ErrNoJobs
	}

	if len(q.pending) == 0 {
		// Set next poll time to avoid busy waiting
		q.nextPoll = time.Now().Add(100 * time.Millisecond)
		return nil, ErrNoJobs
	}

	// Get the first job (highest priority)
	job := q.pending[0]
	q.pending = q.pending[1:]

	// Reset poll time on successful dequeue
	q.nextPoll = time.Time{}

	return job, nil
}

// GetJob retrieves a job by ID
func (q *MemoryQueue) GetJob(ctx context.Context, id string) (*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	job, exists := q.jobs[id]
	if !exists {
		return nil, ErrJobNotFound
	}

	// Return a copy to avoid race conditions
	jobCopy := *job
	return &jobCopy, nil
}

// UpdateJob updates an existing job
func (q *MemoryQueue) UpdateJob(job *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.jobs[job.ID]; !exists {
		return ErrJobNotFound
	}

	q.jobs[job.ID] = job
	return nil
}

// CancelJob cancels a pending job
func (q *MemoryQueue) CancelJob(ctx context.Context, id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, exists := q.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	if job.Status != StatusPending {
		return ErrJobNotFound
	}

	// Remove from pending queue
	for i, pendingJob := range q.pending {
		if pendingJob.ID == id {
			q.pending = append(q.pending[:i], q.pending[i+1:]...)
			break
		}
	}

	// Update job status
	job.Status = StatusCancelled
	now := time.Now()
	job.CompletedAt = &now

	return nil
}

// ListJobs lists jobs with optional status filter
func (q *MemoryQueue) ListJobs(ctx context.Context, status string, limit int) ([]*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	jobs := make([]*Job, 0, len(q.jobs))

	for _, job := range q.jobs {
		if status == "" || job.Status == status {
			// Return a copy to avoid race conditions
			jobCopy := *job
			jobs = append(jobs, &jobCopy)

			if limit > 0 && len(jobs) >= limit {
				break
			}
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})

	return jobs, nil
}

// Size returns the number of pending jobs
func (q *MemoryQueue) Size() (int, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.pending), nil
}

// Clear removes all jobs from the queue
func (q *MemoryQueue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.jobs = make(map[string]*Job)
	q.pending = make([]*Job, 0)
	q.nextPoll = time.Time{}

	return nil
}
