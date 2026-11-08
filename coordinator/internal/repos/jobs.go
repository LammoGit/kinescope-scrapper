package repos

import (
	"coordinator/internal/domain"
	"fmt"
	"sort"
	"sync"
)

var _ IJobRepository = &JobRepository{}

// IJobRepository stores scraping jobs and their state
type IJobRepository interface {
	CreateJob(name string, settings domain.JobSettings) domain.Job
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

// JobRepository is an in-memory job store
type JobRepository struct {
	jobs   map[int]domain.Job
	m      sync.RWMutex
	nextID int
}

// NewJobRepository creates a new JobRepository
func NewJobRepository() *JobRepository {
	return &JobRepository{jobs: make(map[int]domain.Job)}
}

// CreateJob creates and stores a new job
func (jr *JobRepository) CreateJob(name string, settings domain.JobSettings) domain.Job {
	jr.m.Lock()
	defer jr.m.Unlock()

	jr.nextID++
	id := jr.nextID

	if name == "" {
		name = fmt.Sprintf("job-%d", id)
	}

	job := domain.Job{
		Config: domain.JobConfig{
			ID:      id,
			Name:    name,
			Job:     settings,
			Servers: []domain.Worker{},
		},
		Status:  domain.JobStatusCreated,
		Metrics: domain.JobMetrics{},
	}
	jr.jobs[id] = job

	return job
}

// AssignJob attaches a worker to the given job
func (jr *JobRepository) AssignJob(id int, worker *domain.Worker) bool {
	jr.m.Lock()
	defer jr.m.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return false
	}
	if !job.AddWorker(worker) {
		return false
	}
	job.Config.Servers = append(job.Config.Servers, *worker)
	jr.jobs[id] = job
	return true
}

// GetJob returns a job by its identifier
func (jr *JobRepository) GetJob(id int) (domain.Job, bool) {
	jr.m.RLock()
	defer jr.m.RUnlock()

	job, ok := jr.jobs[id]
	return job, ok
}

// GetJobs returns every stored job ordered by identifier
func (jr *JobRepository) GetJobs() []domain.Job {
	jr.m.RLock()
	defer jr.m.RUnlock()

	ids := make([]int, 0, len(jr.jobs))
	for id := range jr.jobs {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	res := make([]domain.Job, 0, len(ids))
	for _, id := range ids {
		res = append(res, jr.jobs[id])
	}
	return res
}

// DeleteJob removes a job
func (jr *JobRepository) DeleteJob(id int) bool {
	jr.m.Lock()
	defer jr.m.Unlock()

	if _, ok := jr.jobs[id]; !ok {
		return false
	}
	delete(jr.jobs, id)
	return true
}

// StartJob moves a job into the running state
func (jr *JobRepository) StartJob(id int) bool {
	jr.m.Lock()
	defer jr.m.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return false
	}

	job.Status = domain.JobStatusRunning
	job.Paused = false
	jr.jobs[id] = job
	return true
}

// PauseJob moves a job into the paused state
func (jr *JobRepository) PauseJob(id int) bool {
	jr.m.Lock()
	defer jr.m.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return false
	}

	job.Status = domain.JobStatusPaused
	job.Paused = true
	jr.jobs[id] = job
	return true
}

// ResumeJob moves a paused job back into the running state
func (jr *JobRepository) ResumeJob(id int) bool {
	jr.m.Lock()
	defer jr.m.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return false
	}

	job.Status = domain.JobStatusRunning
	job.Paused = false
	jr.jobs[id] = job
	return true
}

// StopJob moves a job into the stopped state
func (jr *JobRepository) StopJob(id int) bool {
	jr.m.Lock()
	defer jr.m.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return false
	}

	job.Status = domain.JobStatusStopped
	job.Paused = true
	job.WorkerID = ""
	jr.jobs[id] = job
	return true
}

// UpdateJob replaces a stored job
func (jr *JobRepository) UpdateJob(job domain.Job) bool {
	jr.m.Lock()
	defer jr.m.Unlock()

	if _, ok := jr.jobs[job.Config.ID]; !ok {
		return false
	}

	jr.jobs[job.Config.ID] = job
	return true
}
