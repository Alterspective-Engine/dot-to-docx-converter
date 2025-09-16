package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

const (
	queueKey    = "conversion:queue"
	jobsKey     = "conversion:jobs"
	priorityKey = "conversion:priority"
)

// RedisQueue implements Queue interface using Redis
type RedisQueue struct {
	client *redis.Client
}

// NewRedisQueue creates a new Redis-based queue
func NewRedisQueue(redisURL string) (*RedisQueue, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info("Connected to Redis queue")

	return &RedisQueue{
		client: client,
	}, nil
}

// Enqueue adds a job to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, job *Job) error {
	// Serialize job
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to serialize job: %w", err)
	}

	// Store job data
	if err := q.client.HSet(ctx, jobsKey, job.ID, jobData).Err(); err != nil {
		return fmt.Errorf("failed to store job: %w", err)
	}

	// Add to priority queue
	score := float64(time.Now().Unix()) - float64(job.Priority*1000)
	if err := q.client.ZAdd(ctx, priorityKey, redis.Z{
		Score:  score,
		Member: job.ID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to add job to queue: %w", err)
	}

	log.Debugf("Enqueued job %s with priority %d", job.ID, job.Priority)
	return nil
}

// Dequeue retrieves the next job from the queue
func (q *RedisQueue) Dequeue(ctx context.Context) (*Job, error) {
	// Get highest priority job
	result := q.client.ZPopMin(ctx, priorityKey, 1)
	members, err := result.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue: %w", err)
	}

	if len(members) == 0 {
		return nil, ErrNoJobs
	}

	jobID := members[0].Member.(string)

	// Get job data
	jobData, err := q.client.HGet(ctx, jobsKey, jobID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job data: %w", err)
	}

	// Deserialize job
	var job Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to deserialize job: %w", err)
	}

	log.Debugf("Dequeued job %s", job.ID)
	return &job, nil
}

// GetJob retrieves a job by ID
func (q *RedisQueue) GetJob(ctx context.Context, id string) (*Job, error) {
	jobData, err := q.client.HGet(ctx, jobsKey, id).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	var job Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to deserialize job: %w", err)
	}

	return &job, nil
}

// UpdateJob updates an existing job
func (q *RedisQueue) UpdateJob(job *Job) error {
	ctx := context.Background()

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to serialize job: %w", err)
	}

	if err := q.client.HSet(ctx, jobsKey, job.ID, jobData).Err(); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// CancelJob cancels a pending job
func (q *RedisQueue) CancelJob(ctx context.Context, id string) error {
	// Get the job
	job, err := q.GetJob(ctx, id)
	if err != nil {
		return err
	}

	// Only cancel if pending
	if job.Status != StatusPending {
		return fmt.Errorf("can only cancel pending jobs, current status: %s", job.Status)
	}

	// Remove from priority queue
	if err := q.client.ZRem(ctx, priorityKey, id).Err(); err != nil {
		log.Warnf("Failed to remove job %s from priority queue: %v", id, err)
	}

	// Update status
	job.Status = StatusCancelled
	now := time.Now()
	job.CompletedAt = &now

	return q.UpdateJob(job)
}

// ListJobs lists jobs with optional status filter
func (q *RedisQueue) ListJobs(ctx context.Context, status string, limit int) ([]*Job, error) {
	// Get all job IDs
	jobMap, err := q.client.HGetAll(ctx, jobsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	jobs := make([]*Job, 0, len(jobMap))
	for _, jobData := range jobMap {
		var job Job
		if err := json.Unmarshal([]byte(jobData), &job); err != nil {
			log.Warnf("Failed to deserialize job: %v", err)
			continue
		}

		// Apply status filter
		if status != "" && job.Status != status {
			continue
		}

		jobs = append(jobs, &job)

		// Apply limit
		if limit > 0 && len(jobs) >= limit {
			break
		}
	}

	return jobs, nil
}

// Size returns the number of pending jobs
func (q *RedisQueue) Size() (int, error) {
	ctx := context.Background()
	count, err := q.client.ZCard(ctx, priorityKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue size: %w", err)
	}
	return int(count), nil
}

// Clear removes all jobs from the queue
func (q *RedisQueue) Clear() error {
	ctx := context.Background()

	// Clear all keys
	if err := q.client.Del(ctx, queueKey, jobsKey, priorityKey).Err(); err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	log.Info("Queue cleared")
	return nil
}

// Close closes the Redis connection
func (q *RedisQueue) Close() error {
	return q.client.Close()
}