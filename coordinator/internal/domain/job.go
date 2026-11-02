package domain

type Job struct {
	Config  JobConfig
	Status  JobStatus
	workers []*Worker
}

type JobConfig struct {
	ID      int         `json:"id" validate:"required"`
	Name    string      `json:"name" validate:"required"`
	Job     JobSettings `json:"job" validate:"required"`
	Servers []Worker    `json:"servers"`
}

type JobSettings struct {
	Range                    JobRange `json:"range" validate:"required"`
	RequestTimeout           int      `json:"request_timeout_seconds" validate:"required"`
	MaxRetries               int      `json:"max_retries" validate:"required"`
	GlobalRateLimitRPS       int      `json:"global_rate_limit_rps"`
	StopOnConsecutiveMissing int      `json:"stop_on_consecutive_missing" validate:"required"`
}

type JobRange struct {
	From int `json:"from" validate:"required"`
	To   int `json:"to" validate:"required"`
}

type JobStatus string

const (
	JobStatusCreated    JobStatus = "created"
	JobStatusValidating JobStatus = "validating"
	JobStatusRunning    JobStatus = "running"
	JobStatusPausing    JobStatus = "pausing"
	JobStatusPaused     JobStatus = "paused"
	JobStatusStopping   JobStatus = "stopping"
	JobStatusStopped    JobStatus = "stopped"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

func (j *Job) AddWorker(w *Worker) bool {
	for _, iw := range j.workers {
		if iw.Config.ID == w.Config.ID {
			return false
		}
	}
	j.workers = append(j.workers, w)
	return true
}
