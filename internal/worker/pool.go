package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/api"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/converter"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/queue"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
)

var (
	// Metrics
	jobsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "converter_jobs_processed_total",
			Help: "Total number of conversion jobs processed",
		},
		[]string{"status"},
	)

	jobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "converter_job_duration_seconds",
			Help:    "Duration of conversion jobs in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
		},
		[]string{"status"},
	)

	activeWorkers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "converter_active_workers",
			Help: "Number of active workers",
		},
	)

	queueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "converter_queue_size",
			Help: "Number of jobs in queue",
		},
	)
)

// Pool manages a pool of workers
type Pool struct {
	workerCount int
	queue       queue.Queue
	converter   converter.Converter
	storage     storage.Storage
	wg          sync.WaitGroup
	stopChan    chan struct{}
}

// NewPool creates a new worker pool
func NewPool(workerCount int, q queue.Queue, c converter.Converter, s storage.Storage) *Pool {
	return &Pool{
		workerCount: workerCount,
		queue:       q,
		converter:   c,
		storage:     s,
		stopChan:    make(chan struct{}),
	}
}

// Start begins processing jobs with the worker pool
func (p *Pool) Start(ctx context.Context) {
	log.Infof("Starting worker pool with %d workers", p.workerCount)

	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// Monitor queue size
	go p.monitorQueue(ctx)

	// Wait for all workers to finish
	p.wg.Wait()
	log.Info("Worker pool stopped")
}

// Stop stops the worker pool
func (p *Pool) Stop() {
	close(p.stopChan)
	p.wg.Wait()
}

// worker is the main worker loop
func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	log.Infof("Worker %d started", id)

	activeWorkers.Inc()
	defer activeWorkers.Dec()

	for {
		select {
		case <-ctx.Done():
			log.Infof("Worker %d stopping (context cancelled)", id)
			return
		case <-p.stopChan:
			log.Infof("Worker %d stopping (stop signal)", id)
			return
		default:
			// Try to get a job from the queue
			job, err := p.queue.Dequeue(ctx)
			if err != nil {
				if err == queue.ErrNoJobs {
					// No jobs available, wait a bit
					time.Sleep(1 * time.Second)
					continue
				}
				log.Errorf("Worker %d: Failed to dequeue job: %v", id, err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Process the job
			p.processJob(ctx, id, job)
		}
	}
}

// processJob handles a single conversion job
func (p *Pool) processJob(ctx context.Context, workerID int, job *queue.Job) {
	start := time.Now()
	log.Infof("Worker %d: Processing job %s", workerID, job.ID)

	// Record metrics for active processing
	api.RecordProcessingStart()

	// Update job status
	job.Status = queue.StatusProcessing
	job.StartedAt = &start
	if err := p.queue.UpdateJob(job); err != nil {
		log.Errorf("Failed to update job status: %v", err)
	}

	// Download input file from storage if needed
	localInput, err := p.storage.Download(ctx, job.InputPath)
	if err != nil {
		p.failJob(job, fmt.Errorf("failed to download input: %w", err))
		api.RecordProcessingFailed()
		return
	}
	defer p.storage.Cleanup(localInput)

	// Prepare output path
	localOutput := p.storage.GetLocalPath(job.OutputPath)

	// Perform conversion
	err = p.converter.Convert(ctx, localInput, localOutput)
	if err != nil {
		p.failJob(job, fmt.Errorf("conversion failed: %w", err))
		api.RecordProcessingFailed()
		jobsProcessed.WithLabelValues("failed").Inc()
		jobDuration.WithLabelValues("failed").Observe(time.Since(start).Seconds())
		return
	}

	// Upload output file to storage
	if err := p.storage.Upload(ctx, localOutput, job.OutputPath); err != nil {
		p.failJob(job, fmt.Errorf("failed to upload output: %w", err))
		api.RecordProcessingFailed()
		return
	}

	// Update job as completed
	now := time.Now()
	job.Status = queue.StatusCompleted
	job.CompletedAt = &now
	job.Duration = time.Since(start)

	if err := p.queue.UpdateJob(job); err != nil {
		log.Errorf("Failed to update completed job: %v", err)
	}

	// Record metrics for completed job
	api.RecordProcessingComplete(job.Duration)
	jobsProcessed.WithLabelValues("success").Inc()
	jobDuration.WithLabelValues("success").Observe(time.Since(start).Seconds())

	log.Infof("Worker %d: Completed job %s in %v", workerID, job.ID, job.Duration)
}

// failJob marks a job as failed
func (p *Pool) failJob(job *queue.Job, err error) {
	now := time.Now()
	job.Status = queue.StatusFailed
	job.Error = err.Error()
	job.CompletedAt = &now

	if job.StartedAt != nil {
		job.Duration = now.Sub(*job.StartedAt)
	}

	if err := p.queue.UpdateJob(job); err != nil {
		log.Errorf("Failed to update failed job: %v", err)
	}

	log.Errorf("Job %s failed: %v", job.ID, err)
}

// monitorQueue periodically updates queue size metric
func (p *Pool) monitorQueue(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if size, err := p.queue.Size(); err == nil {
				queueSize.Set(float64(size))
			}
		}
	}
}
