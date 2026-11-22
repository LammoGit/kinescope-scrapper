package services

import (
	"context"
	"coordinator/internal/domain"
	st "coordinator/internal/storage"
	"coordinator/internal/utility"
	"coordinator/pkg/sse"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// Errors
var (
	ErrJobStorageUnavailable = utility.NewInternalError("job storage is not configured")
)

// JobService exposes scraping jobs and their lifecycle
type JobService struct {
	storage    st.Storage
	hub        *sse.Hub
	leasers    map[int]chan domain.Range
	scheduleCh chan struct{}
	leaseTime  time.Duration
	mu         sync.RWMutex
}

// Work estimate used for open-ended jobs, they never run out of identifiers
const openRangeRemaining = 1 << 30

// NewJobService creates a new JobService
func NewJobService(ctx context.Context, hub *sse.Hub, storage st.Storage) (*JobService, error) {
	if storage == nil {
		return nil, ErrJobStorageUnavailable
	}
	s := &JobService{
		storage:    storage,
		hub:        hub,
		leasers:    make(map[int]chan domain.Range),
		scheduleCh: make(chan struct{}, 1),
		leaseTime:  2 * time.Minute,
	}
	go s.run(ctx)
	return s, nil
}

// CreateJob creates a job together with the ranges it processes
func (s *JobService) CreateJob(ctx context.Context, name string, config domain.JobConfig) (domain.Job, error) {
	job, err := s.storage.CreateJob(ctx, name, config)
	result := "success"
	code := 200
	msg := "job is created successfully"
	if err != nil {
		result = "fail"
		code = utility.StatusFromError(err)
		msg = err.Error()
	} else {
		s.publish("job.create", job)
		slog.Info("Job created", "jobID", job.ID, "name", job.Config.Name)
	}
	s.record(ctx, job.ID, "create", result, code, msg)
	return job, err
}

// GetJobs returns every job
func (s *JobService) GetJobs(ctx context.Context) ([]domain.Job, error) {
	return s.storage.GetJobs(ctx)
}

// GetJobByID returns a job with the given id
func (s *JobService) GetJobByID(ctx context.Context, id int) (domain.Job, error) {
	return s.storage.GetJob(ctx, id)
}

// ReplaceJobByID replaces previous job with new one
func (s *JobService) ReplaceJobByID(ctx context.Context, job domain.Job) error {
	err := s.storage.ReplaceJob(ctx, job)
	result := "success"
	code := 200
	msg := "job is updated successfully"
	if err != nil {
		result = "fail"
		code = utility.StatusFromError(err)
		msg = err.Error()
	} else {
		s.publish("job.update", job)
		slog.Info("Job updated", "jobID", job.ID)
	}
	s.record(ctx, job.ID, "update", result, code, msg)
	return err
}

// DeleteJob removes a job and its ranges
func (s *JobService) DeleteJob(ctx context.Context, id int) error {
	job, err := s.storage.DeleteJob(ctx, id)
	result := "success"
	code := 200
	msg := "job is deleted successfully"
	if err != nil {
		result = "fail"
		code = utility.StatusFromError(err)
		msg = err.Error()
	} else if s.hub != nil {
		slog.Info("Job deleted", "jobID", id)
		msg := sse.SSEMessage{
			Event: "job.delete",
			Data:  map[string]any{"job_id": id},
		}

		var topics []string
		topics = append(topics, "job-*", "worker-*")
		topics = append(topics, fmt.Sprintf("job-%d", id))
		topics = append(topics, fmt.Sprintf("job-update-%d", id), "job-update-*")
		for _, id := range job.Workers {
			topics = append(topics, fmt.Sprintf("worker-%d", id))
		}
		s.hub.PublishAll(topics, msg)
	}
	s.record(ctx, id, "delete", result, code, msg)
	return nil
}

// StartJob moves a job into the running state
func (s *JobService) StartJob(ctx context.Context, id int) error {
	return s.changeState(ctx, id, "start", "job is started successfully", domain.JobStatusRunning, s.storage.StartJob)
}

// PauseJob moves a job into the paused state
func (s *JobService) PauseJob(ctx context.Context, id int) error {
	return s.changeState(ctx, id, "pause", "job is paused successfully", domain.JobStatusPaused, s.storage.PauseJob)
}

// ResumeJob moves a paused job back into the running state
func (s *JobService) ResumeJob(ctx context.Context, id int) error {
	return s.changeState(ctx, id, "resume", "job is resumed successfully", domain.JobStatusRunning, s.storage.ResumeJob)
}

// StopJob moves a job into the stopped state and releases its ranges
func (s *JobService) StopJob(ctx context.Context, id int) error {
	// publish to workers that they are deassigned from the job
	err := s.changeState(ctx, id, "stop", "job is stopped successfully", domain.JobStatusStopped, s.storage.StopJob)
	if err != nil {
		return err
	}
	s.storage.ReleaseRangesByJobID(ctx, id)
	return nil
}

// changeState changes state of a job, records event, and publishes state change message to the hub
func (s *JobService) changeState(ctx context.Context, id int, operation, message, state string, change func(context.Context, int) (domain.Job, error)) error {
	job, err := change(ctx, id)
	if err == nil {
		slog.Info("Job state changed", "jobID", id, "operation", operation, "state", state)
	}

	// Log state change event
	result := "success"
	code := 200
	msg := message
	if err != nil {
		result = "fail"
		code = utility.StatusFromError(err)
		msg = err.Error()
	}
	s.record(ctx, id, operation, result, code, msg)

	// Send job state update message
	if s.hub == nil {
		return err
	}

	var topics []string
	topics = append(topics, "job-*", "worker-*")
	topics = append(topics, fmt.Sprintf("job-%d", id))
	topics = append(topics, fmt.Sprintf("job-update-%d", id), "job-update-*")
	for _, id := range job.Workers {
		topics = append(topics, fmt.Sprintf("worker-%d", id))
	}
	s.hub.PublishAll(topics, sse.SSEMessage{Event: "job.state_change", Data: map[string]any{
		"job_id": id,
		"state":  state,
	}})

	return nil
}

// publish publishes job event to the hub
func (s *JobService) publish(event string, job domain.Job) {
	if s.hub == nil {
		return
	}
	topic := fmt.Sprintf("job-%d", job.ID)
	s.hub.PublishAll([]string{topic, "job-*"}, sse.SSEMessage{Event: event, Data: job})
	// Notify subscribers of /stream/jobs/updates about the change
	s.hub.PublishAll(
		[]string{fmt.Sprintf("job-update-%d", job.ID), "job-update-*"},
		sse.SSEMessage{Event: event, Data: job},
	)
}

// record logs job event and publishes message to the hub
func (s *JobService) record(ctx context.Context, id int, operation string, result string, code int, message string) {
	event := domain.Event{
		Time:       time.Now(),
		Level:      domain.EventLevelInfo,
		Component:  "Job service",
		JobID:      id,
		WorkerID:   0,
		RangeID:    0,
		ResourceID: 0,
		Operation:  operation,
		Result:     result,
		ErrorCode:  code,
		Message:    message,
	}
	stored := s.storage.AddEvent(ctx, event)
	if s.hub != nil {
		s.hub.PublishAll(
			[]string{"events-*", fmt.Sprintf("events-job-%d", id)},
			sse.SSEMessage{Event: "event", Data: stored},
		)
	}
}

// run waits for scheduling requests and assigns available ranges
func (s *JobService) run(ctx context.Context) {
	for {
		select {
		case <-s.scheduleCh:
			s.assignAvailableRanges(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// signal asks the scheduling loop to run without blocking the caller
func (s *JobService) signal() {
	select {
	case s.scheduleCh <- struct{}{}:
	default:
	}
}

// assignAvailableRanges leases free ranges to workers waiting for one
func (s *JobService) assignAvailableRanges(ctx context.Context) {
	if err := ctx.Err(); err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.leasers) == 0 {
		return
	}

	jobs, err := s.storage.GetJobs(ctx)
	if err != nil {
		return
	}

	type candidate struct {
		id       int
		finishIn time.Duration
	}
	candidates := make([]candidate, 0, len(jobs))
	for _, job := range jobs {
		if job.Status != domain.JobStatusRunning || job.Paused {
			continue
		}

		ranges, err := s.storage.GetRangesByJobID(ctx, job.ID)
		if err != nil {
			continue
		}

		remaining := 0
		frontier := job.Config.Range.From - 1
		open := job.Config.Range.To <= 0
		for _, r := range ranges {
			if r.RangeEnd > frontier {
				frontier = r.RangeEnd
			}
			if r.Status == domain.JobStatusCompleted || r.Exhausted() {
				continue
			}
			remaining += r.RangeEnd - r.NextID + 1
		}
		if open {
			remaining += openRangeRemaining
		} else if frontier < job.Config.Range.To {
			remaining += job.Config.Range.To - frontier
		}
		if remaining <= 0 {
			continue
		}

		workers := len(job.Workers)
		if workers == 0 {
			workers = 1
		}
		var ttf time.Duration
		if open {
			ttf = time.Duration(1<<62 - 1)
		} else if m, err := s.storage.GetJobMetricsByID(ctx, job.ID); err == nil && m.IdentifiersPerSecond > 0 {
			perSec := m.IdentifiersPerSecond * float64(workers)
			ttf = time.Duration(float64(remaining) / perSec * float64(time.Second))
		} else {
			ttf = time.Duration(1<<62 - 1)
		}
		candidates = append(candidates, candidate{job.ID, ttf})
	}
	if len(candidates) == 0 {
		return
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].finishIn != candidates[j].finishIn {
			return candidates[i].finishIn > candidates[j].finishIn
		}
		return candidates[i].id < candidates[j].id
	})

	workerIDs := make([]int, 0, len(s.leasers))
	for id := range s.leasers {
		workerIDs = append(workerIDs, id)
	}
	sort.Ints(workerIDs)

	const (
		loadThreshold  = 0.85
		queueThreshold = 10
		minFreeSpace   = 1 << 30
	)

	for _, workerID := range workerIDs {
		ch := s.leasers[workerID]

		if m, err := s.storage.GetWorkerMetricsByID(ctx, workerID); err == nil {
			if m.CPULoad > loadThreshold || m.MemoryLoad > loadThreshold ||
				m.DiskLoad > loadThreshold || m.NetworkLoad > loadThreshold ||
				m.DownloadQueueSize > queueThreshold ||
				(m.FreeSpaceBytes > 0 && m.FreeSpaceBytes < minFreeSpace) {
				continue
			}
		}

		var claimed domain.Range
		var got bool
		for _, c := range candidates {
			r, err := s.storage.TryClaimRange(ctx, workerID, c.id, s.leaseTime)
			if err != nil {
				continue
			}
			claimed = r
			got = true
			break
		}
		if !got {
			continue
		}

		select {
		case ch <- claimed:
			delete(s.leasers, workerID)
		default:
			_ = s.storage.ReleaseRangeByID(ctx, claimed.ID)
		}
	}

	if len(s.leasers) > 0 {
		s.signal()
	}
}

// ReceiveWorkerHeartbeat validates that the worker is still registered
func (s *JobService) ReceiveWorkerHeartbeat(ctx context.Context, workerID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.storage.RenewLeases(ctx, workerID, s.leaseTime)
	return nil
}

// Lease blocks until a range is assigned to the worker or the context ends
func (s *JobService) Lease(ctx context.Context, workerID int) (domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return domain.Range{}, err
	}

	s.mu.Lock()
	var ch chan domain.Range
	if chl, ok := s.leasers[workerID]; ok {
		ch = chl
	} else {
		ch = make(chan domain.Range, 1)
		s.leasers[workerID] = ch
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if current, ok := s.leasers[workerID]; ok && current == ch {
			delete(s.leasers, workerID)
		}
		s.mu.Unlock()
	}()

	s.signal()

	select {
	case r := <-ch:
		return r, nil
	case <-ctx.Done():
		return domain.Range{}, ctx.Err()
	}
}

// WorkerCompleted marks a range as fully processed by its worker
func (s *JobService) WorkerCompleted(ctx context.Context, workerID, rangeID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r, err := s.storage.GetRangeByID(ctx, rangeID)
	if err != nil {
		return err
	}
	if r.LeasedBy != 0 && r.LeasedBy != workerID {
		return utility.NewForbiddenError("range is leased by another worker")
	}

	if err := s.storage.CompleteRange(ctx, rangeID); err != nil {
		return err
	}

	s.signal()
	return nil
}

// FailedLink records a warning event about a link the worker failed to process
func (s *JobService) FailedLink(ctx context.Context, workerID int, url, msg string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := s.storage.GetWorkerByID(ctx, workerID); err != nil {
		return err
	}

	return s.storage.AddEvent(ctx, domain.Event{
		Time:      time.Now(),
		Level:     domain.EventLevelWarning,
		Component: "scheduler",
		WorkerID:  workerID,
		Operation: "download",
		Result:    "failed",
		Message:   fmt.Sprintf("failed to process %s: %s", url, msg),
	})
}
