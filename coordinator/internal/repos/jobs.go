package repos

import (
	"coordinator/internal/domain"
	"fmt"
	"sync"
	"time"
)

var _ IJobRepository = &JobRepository{}

type IJobRepository interface {
	CreateJob(settings domain.JobSettings) domain.Job
	AssignJob(id int, worker *domain.Worker) bool
	GetJob(id int) (domain.Job, bool)
	GetJobs() []domain.Job
	DeleteJob(id int) bool
	StartJob(id int) bool
	PauseJob(id int) bool
	ResumeJob(id int) bool
	StopJob(id int) bool
	UpdateJob(job domain.Job) bool
}

type JobRepository struct {
	jobs   map[int]domain.Job
	m      sync.RWMutex
	nextID int
}

// CreateJob creates and stores a new job
func (jr *JobRepository) CreateJob(settings domain.JobSettings) domain.Job {
	jr.m.Lock()
	defer jr.m.Unlock()

	jr.nextID++
	id := jr.nextID

	jobName := fmt.Sprintf("job-%d", id)
	jr.jobs[id] = domain.Job{
		Config: domain.JobConfig{
			ID:      id,
			Name:    jobName,
			Job:     settings,
			Servers: []domain.Worker{},
		},
		Status:  domain.JobStatusCreated,
		Metrics: domain.JobMetrics{},
	}

	return jr.jobs[id]
}

// AssignJob clears and assigns job to the given worker
func (jr *JobRepository) AssignJob(id int, worker *domain.Worker) bool {
	jr.m.Lock()
	defer jr.m.Unlock()

	// Find a job by its id
	job, ok := jr.jobs[id]
	if !ok {
		return false
	}

	// Assign worker to the job
	return job.AddWorker(worker)
}

// GetJob
func (jr *JobRepository) GetJob(id int) (domain.Job, bool) {
	jr.m.RLock()
	defer jr.m.RUnlock()

	job, ok := jr.jobs[id]
	return job, ok
}

// GetJobs
func (jr *JobRepository) GetJobs() []domain.Job {
	jr.m.RLock()
	defer jr.m.RUnlock()
	var res []domain.Job
	for _, v := range jr.jobs {
		res = append(res, v)
	}
	return res
}

// StartJob
func (jr *JobRepository) StartJob(id int) bool {
	jr.m.Lock()
	defer jr.m.Unlock()
	job, ok := jr.jobs[id]
	if !ok {
		return false
	}

	if job.WorkerID != "" {
		return
	}

	job.Paused = false
	return true
}

// PauseJob
func (jr *JobRepository) PauseJob(id int) bool {
	jr.m.Lock()
	defer jr.m.Unlock()
	job, ok := jr.jobs[id]
	if !ok {
		return
	}

	job.Paused = true
}

// ResumeJob
func (jr *JobRepository) ResumeJob(id int) bool {
	jr.m.Lock()
	defer jr.m.Unlock()
	job, ok := jr.jobs[id]
	if !ok {
		return
	}

	job.Paused = false
	job.Expiration = time.Now().Add(10 * time.Second)
}

// StopJob
func (jr *JobRepository) StopJob(id int) bool {
	jr.m.Lock()
	defer jr.m.Unlock()
	job, ok := jr.jobs[id]
	if !ok {
		return
	}

	if job.WorkerID == "" {
		return
	}

	job.WorkerID = ""
	job.Paused = true
}

// UpdateJob
func (jr *JobRepository) UpdateJob(job domain.Job) bool {
	jr.m.Lock()
	defer jr.m.Unlock()

	oldJob, ok := jr.jobs[job.Config.ID]
	if !ok {
		return false
	}

	jr.jobs[job.Config.ID] = oldJob
	return true
}
