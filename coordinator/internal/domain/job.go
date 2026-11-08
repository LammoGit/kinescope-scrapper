package domain

import "time"

// Job is a scraping task and its current state
type Job struct {
	ID         int       `json:"id" form:"id" binding:"min=0"`
	Config     JobConfig `json:"config" form:"config" binding:"required"`
	Status     string    `json:"status" form:"status" binding:"required,oneof=created validating running pausing paused stopping stopped completed failed"`
	Metrics    Metrics   `json:"metrics" form:"metrics"`
	WorkerID   string    `json:"worker_id" form:"worker_id"`
	Paused     bool      `json:"paused" form:"paused"`
	Expiration time.Time `json:"expiration" form:"expiration"`
	workers    []*Worker
}

// JobCreate is the payload accepted when creating a new job
type JobCreate struct {
	Name string      `json:"name" form:"name" binding:"required,min=1,max=128"`
	Job  JobSettings `json:"job" form:"job" binding:"required"`
}

// JobUpdate optional fields to be updated
type JobUpdate struct {
	Name *string      `json:"name" form:"name" binding:"omitempty,min=1,max=128"`
	Job  *JobSettings `json:"job" form:"job"`
}

// JobConfig contains the user-provided configuration of a job
type JobConfig struct {
	Name    string      `json:"name" form:"name" binding:"required"`
	Job     JobSettings `json:"job" form:"job" binding:"required"`
	Servers []Worker    `json:"servers" form:"servers"`
}

// JobSettings contains the processing parameters of a job
type JobSettings struct {
	Range                    JobRange `json:"range" form:"range" binding:"required"`
	RequestTimeout           int      `json:"request_timeout_seconds" form:"request_timeout_seconds" binding:"required,min=1"`
	MaxRetries               int      `json:"max_retries" form:"max_retries" binding:"min=0"`
	GlobalRateLimitRPS       int      `json:"global_rate_limit_rps" form:"global_rate_limit_rps" binding:"min=0"`
	StopOnConsecutiveMissing int      `json:"stop_on_consecutive_missing" form:"stop_on_consecutive_missing" binding:"min=0"`
}

// JobRange is the inclusive range of identifiers a job processes
type JobRange struct {
	From int `json:"from" form:"from" binding:"required,min=0"`
	To   int `json:"to" form:"to" binding:"required,min=0,gtefield=From"`
}

const (
	JobStatusCreated    string = "created"
	JobStatusValidating string = "validating"
	JobStatusRunning    string = "running"
	JobStatusPausing    string = "pausing"
	JobStatusPaused     string = "paused"
	JobStatusStopping   string = "stopping"
	JobStatusStopped    string = "stopped"
	JobStatusCompleted  string = "completed"
	JobStatusFailed     string = "failed"
)

// Update applies the non-nil fields of an update to a job
func (j *Job) Update(u JobUpdate) {
	if u.Name != nil {
		j.Config.Name = *u.Name
	}
	if u.Job != nil {
		j.Config.Job = *u.Job
	}
}

// AddWorker attaches a worker to the job
// It returns false when the worker is already attached
func (j *Job) AddWorker(w *Worker) bool {
	for _, iw := range j.workers {
		if iw.WorkerID() == w.WorkerID() {
			return false
		}
	}
	j.workers = append(j.workers, w)
	return true
}

// RemoveWorker detaches a worker from the job
// It returns false when the worker wasn't attached
func (j *Job) RemoveWorker(id int) bool {
	for i, iw := range j.workers {
		if iw.WorkerID() == id {
			j.workers = append(j.workers[:i], j.workers[i+1:]...)
			return true
		}
	}
	return false
}
