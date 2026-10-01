package job_distributor

import (
	st "coordinator/internal/storage"
	"fmt"
	"sync"
	"time"
)

type JobDistributor struct {
	mu   sync.RWMutex
	jobs map[string]st.Job
}

func New() *JobDistributor {
	jd := &JobDistributor{
		jobs: make(map[string]st.Job),
	}
	jd.Run()
	return jd
}

func (jd *JobDistributor) Run() {

}

func (jd *JobDistributor) CreateJob(from, to uint) (jobID string) {
	jd.mu.Lock()
	defer jd.mu.Unlock()

	for i := 1; ; i++ {
		name := fmt.Sprintf("job-%05d", i)
		if _, ok := jd.jobs[name]; !ok {
			jobID = name
			break
		}
	}

	issue := time.Now()
	jd.jobs[jobID] = st.Job{
		JobID:      jobID,
		Issue:      issue,
		Expiration: issue.Add(10 * time.Second),
		From:       from,
		To:         to,
		Progress:   from,
		Attempt:    0,
		Status:     "idle",
		Paused:     false,
	}

	return
}

func (jd *JobDistributor) Heartbeat(jobID, workerID string) bool {
	jd.mu.Lock()
	defer jd.mu.Unlock()

	job, ok := jd.jobs[jobID]
	if !ok || job.WorkerID != workerID {
		return false
	}

	job.Expiration = time.Now().Add(10 * time.Second)
	return true
}

func (jd *JobDistributor) AssignJob(jobID, workerID string) bool {
	jd.mu.Lock()
	defer jd.mu.Unlock()

	job, ok := jd.jobs[jobID]
	job.WorkerID = workerID
	job.Status = "working"
	return ok
}

func (jd *JobDistributor) GetJob(jobID string) (st.Job, bool) {
	jd.mu.RLock()
	defer jd.mu.RUnlock()

	job, ok := jd.jobs[jobID]
	return job, ok
}

func (jd *JobDistributor) GetJobs() map[string]st.Job {
	jd.mu.RLock()
	defer jd.mu.RUnlock()

	out := make(map[string]st.Job, len(jd.jobs))
	for k, v := range jd.jobs {
		out[k] = v
	}
	return out
}

func (jd *JobDistributor) StartJob(jobID, workerID string) {
	jd.mu.Lock()
	defer jd.mu.Unlock()
	job, ok := jd.jobs[jobID]
	if !ok {
		return
	}

	if job.WorkerID != "" {
		return
	}

	job.Issue = time.Now()
	job.Expiration = job.Issue.Add(10 * time.Second)
	job.WorkerID = workerID
	job.Progress = job.From
	job.Paused = false
}

func (jd *JobDistributor) PauseJob(jobID string) {
	jd.mu.Lock()
	defer jd.mu.Unlock()
	job, ok := jd.jobs[jobID]
	if !ok {
		return
	}

	job.Paused = true
}

func (jd *JobDistributor) ResumeJob(jobID string) {
	jd.mu.Lock()
	defer jd.mu.Unlock()
	job, ok := jd.jobs[jobID]
	if !ok {
		return
	}

	job.Paused = false
	job.Expiration = time.Now().Add(10 * time.Second)
}

func (jd *JobDistributor) StopJob(jobID string) {
	jd.mu.Lock()
	defer jd.mu.Unlock()
	job, ok := jd.jobs[jobID]
	if !ok {
		return
	}

	if job.WorkerID == "" {
		return
	}

	job.WorkerID = ""
	job.Paused = true
}

func (jd *JobDistributor) JobItems(jobID string) []uint {
	jd.mu.Lock()
	defer jd.mu.Unlock()
	job, ok := jd.jobs[jobID]
	if !ok {
		return []uint{}
	}
	return job.Items
}

func (jd *JobDistributor) JobEvents(jobID string) []string {
	jd.mu.Lock()
	defer jd.mu.Unlock()
	job, ok := jd.jobs[jobID]
	if !ok {
		return []string{}
	}
	return job.Events
}
