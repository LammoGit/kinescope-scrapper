//go:build postgres && !sqlite

package storage

import (
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"testing"
	"time"

	"coordinator/internal/domain"
	"coordinator/internal/repos"
)

// TestPostgresStorage is an integration test. It runs only when
// TEST_DATABASE_URL points to a disposable PostgreSQL database, for example:
//
//	TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/coordinator_test?sslmode=disable \
//	  go test -tags postgres ./internal/storage
func TestPostgresStorage(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	t.Setenv("DATABASE_URL", dsn)

	ctx := context.Background()
	raw, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	p := raw.(*Postgres)
	t.Cleanup(func() { _ = p.Close() })

	if _, err := p.db.ExecContext(ctx,
		`TRUNCATE users, workers, jobs, ranges, videos, events, telemetry RESTART IDENTITY CASCADE`,
	); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	// Users
	user, err := p.CreateUser(ctx, "tester", "tester@example.com",
		domain.PasswordCredentials{Hash: []byte("hash"), Salt: []byte("salt")})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := p.CreateUser(ctx, "other", "tester@example.com",
		domain.PasswordCredentials{Hash: []byte("hash"), Salt: []byte("salt")}); !errors.Is(err, repos.ErrEmailIsTaken) {
		t.Fatalf("expected ErrEmailIsTaken, got %v", err)
	}
	if _, err := p.GetUserByEmail(ctx, user.Email); err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if _, err := p.GetUserByID(ctx, user.ID); err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if _, err := p.GetUserByID(ctx, 999); !errors.Is(err, repos.ErrUserNotFoundByID) {
		t.Fatalf("expected ErrUserNotFoundByID, got %v", err)
	}
	if _, err := p.GetUserByEmail(ctx, "missing@example.com"); !errors.Is(err, repos.ErrUserNotFoundByEmail) {
		t.Fatalf("expected ErrUserNotFoundByEmail, got %v", err)
	}
	user.Username = "renamed"
	if err := p.ReplaceUser(ctx, user); err != nil {
		t.Fatalf("ReplaceUser: %v", err)
	}
	storedUser, err := p.GetUserByID(ctx, user.ID)
	if err != nil || storedUser.Username != "renamed" {
		t.Fatalf("GetUserByID after replace: %v, user=%+v", err, storedUser)
	}

	// Workers
	worker := domain.Worker{
		Config: domain.WorkerConfig{
			Name: "worker", Host: "127.0.0.1", SSHPort: 22, SSHUser: "root",
			Threads: 4, StoragePath: "/tmp/videos", MinFreeSpaceGB: 10,
		},
		Status:    domain.WorkerStatusOff,
		PublicKey: ed25519.PublicKey("public-key"),
	}
	if err := p.AddWorker(ctx, worker); err != nil {
		t.Fatalf("AddWorker: %v", err)
	}
	workers, err := p.GetWorkers(ctx)
	if err != nil || len(workers) != 1 {
		t.Fatalf("GetWorkers: %v, len=%d", err, len(workers))
	}
	worker = workers[0]
	if _, err := p.GetWorkerByID(ctx, worker.ID); err != nil {
		t.Fatalf("GetWorkerByID: %v", err)
	}

	worker.Status = domain.WorkerStatusWorking
	worker.Config.Threads = 8
	if err := p.ReplaceWorker(ctx, worker); err != nil {
		t.Fatalf("ReplaceWorker: %v", err)
	}
	storedWorker, err := p.GetWorkerByID(ctx, worker.ID)
	if err != nil || storedWorker.Status != domain.WorkerStatusWorking || storedWorker.Config.Threads != 8 {
		t.Fatalf("GetWorkerByID after replace: %v, worker=%+v", err, storedWorker)
	}

	explicitWorker := worker
	explicitWorker.ID = 50
	explicitWorker.Config.Name = "explicit-worker"
	if err := p.AddWorker(ctx, explicitWorker); err != nil {
		t.Fatalf("AddWorker explicit id: %v", err)
	}
	if err := p.DeleteWorkerByID(ctx, explicitWorker.ID); err != nil {
		t.Fatalf("DeleteWorkerByID: %v", err)
	}
	if _, err := p.GetWorkerByID(ctx, explicitWorker.ID); !errors.Is(err, repos.ErrWorkerNotFoundByID) {
		t.Fatalf("expected ErrWorkerNotFoundByID, got %v", err)
	}
	if err := p.DeleteWorkerByID(ctx, 999); !errors.Is(err, repos.ErrWorkerNotFoundByID) {
		t.Fatalf("expected ErrWorkerNotFoundByID, got %v", err)
	}

	// Jobs
	job, err := p.CreateJob(ctx, "job-test", domain.JobConfig{
		Name:    "job-test",
		Range:   domain.JobRange{From: 10, To: 20},
		Servers: []domain.WorkerConfig{worker.Config},
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if job.Config.Name != "job-test" {
		t.Fatalf("unexpected job name %q", job.Config.Name)
	}
	jobs, err := p.GetJobs(ctx)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("GetJobs: %v, len=%d", err, len(jobs))
	}
	job.Config.Name = "job-renamed"
	job.Config.MaxTries = 5
	if err := p.ReplaceJob(ctx, job); err != nil {
		t.Fatalf("ReplaceJob: %v", err)
	}
	job, err = p.GetJob(ctx, job.ID)
	if err != nil || job.Config.Name != "job-renamed" || job.Config.MaxTries != 5 {
		t.Fatalf("GetJob after replace: %v, job=%+v", err, job)
	}

	if _, err := p.StartJob(ctx, job.ID); err != nil {
		t.Fatalf("StartJob: %v", err)
	}
	if err := p.AssignJob(ctx, job.ID, worker); err != nil {
		t.Fatalf("AssignJob: %v", err)
	}
	job, err = p.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if job.Status != domain.JobStatusRunning || len(job.Workers) != 1 || job.Workers[0] != worker.ID {
		t.Fatalf("unexpected job: %+v", job)
	}
	if _, err := p.PauseJob(ctx, job.ID); err != nil {
		t.Fatalf("PauseJob: %v", err)
	}
	job, _ = p.GetJob(ctx, job.ID)
	if job.Status != domain.JobStatusPaused || !job.Paused {
		t.Fatalf("unexpected paused job: %+v", job)
	}
	if _, err := p.ResumeJob(ctx, job.ID); err != nil {
		t.Fatalf("ResumeJob: %v", err)
	}
	job, _ = p.GetJob(ctx, job.ID)
	if job.Status != domain.JobStatusRunning || job.Paused {
		t.Fatalf("unexpected resumed job: %+v", job)
	}

	// Ranges and lease lifecycle
	if err := p.AddRange(ctx, domain.Range{
		JobID: job.ID, RangeStart: 10, RangeEnd: 20,
		Status: domain.JobStatusCreated, LeasedBy: 0,
	}); err != nil {
		t.Fatalf("AddRange: %v", err)
	}
	ranges, err := p.GetRangesByJobID(ctx, job.ID)
	if err != nil || len(ranges) != 1 {
		t.Fatalf("GetRangesByJobID: %v, len=%d", err, len(ranges))
	}
	pending, err := p.GetUnassignedRanges(ctx)
	if err != nil || len(pending) != 1 {
		t.Fatalf("GetUnassignedRanges: %v, len=%d", err, len(pending))
	}
	r := pending[0]

	if err := p.TryClaimRange(ctx, r.ID, worker.ID, time.Minute); err != nil {
		t.Fatalf("TryClaimRange: %v", err)
	}
	if err := p.TryClaimRange(ctx, r.ID, worker.ID, time.Minute); !errors.Is(err, repos.ErrRangeIsTaken) {
		t.Fatalf("expected ErrRangeIsTaken, got %v", err)
	}
	if renewed, err := p.RenewLeases(ctx, worker.ID, 2*time.Minute); err != nil || renewed != 1 {
		t.Fatalf("RenewLeases: %v, renewed=%d", err, renewed)
	}
	if err := p.CompleteRange(ctx, r.ID); err != nil {
		t.Fatalf("CompleteRange: %v", err)
	}
	r, err = p.GetRangeByID(ctx, r.ID)
	if err != nil {
		t.Fatalf("GetRangeByID: %v", err)
	}
	if r.Status != domain.JobStatusCompleted || r.LeasedBy != 0 {
		t.Fatalf("unexpected range after complete: %+v", r)
	}
	if err := p.TryClaimRange(ctx, r.ID, worker.ID, time.Minute); !errors.Is(err, repos.ErrRangeIsCompleted) {
		t.Fatalf("expected ErrRangeIsCompleted, got %v", err)
	}

	if err := p.ReleaseRangeByID(ctx, r.ID); err != nil {
		t.Fatalf("ReleaseRangeByID: %v", err)
	}
	r, _ = p.GetRangeByID(ctx, r.ID)
	if r.Status != domain.JobStatusCreated || r.LeasedBy != 0 {
		t.Fatalf("unexpected released range: %+v", r)
	}

	if err := p.AddRange(ctx, domain.Range{
		JobID: job.ID, RangeStart: 21, RangeEnd: 30,
		Status: domain.JobStatusCreated,
	}); err != nil {
		t.Fatalf("AddRange second: %v", err)
	}
	ranges, _ = p.GetRangesByJobID(ctx, job.ID)
	second := ranges[len(ranges)-1]
	if err := p.TryClaimRange(ctx, second.ID, worker.ID, time.Minute); err != nil {
		t.Fatalf("TryClaimRange second: %v", err)
	}
	if err := p.ReleaseRangesByJobID(ctx, job.ID); err != nil {
		t.Fatalf("ReleaseRangesByJobID: %v", err)
	}
	second, _ = p.GetRangeByID(ctx, second.ID)
	if second.LeasedBy != 0 || second.Status != domain.JobStatusCreated {
		t.Fatalf("unexpected range after job release: %+v", second)
	}

	if err := p.DeleteRangesByJobID(ctx, job.ID); err != nil {
		t.Fatalf("DeleteRangesByJobID: %v", err)
	}
	if ranges, _ := p.GetRangesByJobID(ctx, job.ID); len(ranges) != 0 {
		t.Fatalf("ranges were not deleted: %d", len(ranges))
	}

	// Keep one range for the cascade check at the end.
	if err := p.AddRange(ctx, domain.Range{
		JobID: job.ID, RangeStart: 31, RangeEnd: 40,
		Status: domain.JobStatusCreated,
	}); err != nil {
		t.Fatalf("AddRange cascade: %v", err)
	}
	ranges, _ = p.GetRangesByJobID(ctx, job.ID)
	cascadeRange := ranges[0]

	// Events
	if err := p.AddEvent(ctx, domain.Event{
		Time: time.Now(), Level: domain.EventLevelInfo, Component: "test",
		JobID: job.ID, WorkerID: worker.ID, Operation: "create", Result: "success",
		Message: "created",
	}); err != nil {
		t.Fatalf("AddEvent: %v", err)
	}
	if err := p.AddEvent(ctx, domain.Event{
		Time: time.Now(), Level: domain.EventLevelWarning, Component: "test",
		WorkerID: worker.ID, Operation: "download", Result: "failed",
		Message: "failed",
	}); err != nil {
		t.Fatalf("AddEvent second: %v", err)
	}
	events, err := p.GetEvents(ctx)
	if err != nil || len(events) != 2 {
		t.Fatalf("GetEvents: %v, len=%d", err, len(events))
	}
	if events, err := p.GetEventsByJobID(ctx, job.ID); err != nil || len(events) != 1 {
		t.Fatalf("GetEventsByJobID: %v, len=%d", err, len(events))
	}
	if events, err := p.GetEventsByWorkerID(ctx, worker.ID); err != nil || len(events) != 2 {
		t.Fatalf("GetEventsByWorkerID: %v, len=%d", err, len(events))
	}
	var eventID int
	if err := p.db.QueryRowContext(ctx, `SELECT MIN(id) FROM events`).Scan(&eventID); err != nil {
		t.Fatalf("select event id: %v", err)
	}
	if _, err := p.GetEventByID(ctx, eventID); err != nil {
		t.Fatalf("GetEventByID: %v", err)
	}
	if _, err := p.GetEventByID(ctx, 999); !errors.Is(err, repos.ErrEventNotFoundByID) {
		t.Fatalf("expected ErrEventNotFoundByID, got %v", err)
	}

	// Videos
	if err := p.AddVideo(ctx, domain.Video{
		JobID: job.ID,
		Metadata: domain.VideoMetadata{
			Title: "Example video", SourceURL: "https://kinescope.io/1",
			DownloadURL: "https://cdn.example/1.mp4", FoundAt: time.Now(),
			DownloadedAt: time.Now(), FileSize: 1024, MimeType: "video/mp4",
			Status: "downloaded",
		},
		StorageMetadata: domain.VideoStorageMetadata{
			BucketName: "videos", ObjectKey: "1.mp4", Checksum: "abc",
		},
	}); err != nil {
		t.Fatalf("AddVideo: %v", err)
	}
	videos, err := p.GetVideos(ctx, 10, 0)
	if err != nil || len(videos) != 1 {
		t.Fatalf("GetVideos: %v, len=%d", err, len(videos))
	}
	video := videos[0]
	if _, err := p.GetVideoByID(ctx, video.ID); err != nil {
		t.Fatalf("GetVideoByID: %v", err)
	}
	if videos, err := p.GetVideosByJobID(ctx, job.ID); err != nil || len(videos) != 1 {
		t.Fatalf("GetVideosByJobID: %v, len=%d", err, len(videos))
	}
	if videos, err := p.GetVideosByQuery(ctx, "Example", 10, 0); err != nil || len(videos) != 1 {
		t.Fatalf("GetVideosByQuery: %v, len=%d", err, len(videos))
	}
	if _, err := p.GetVideoByID(ctx, 999); !errors.Is(err, repos.ErrVideoNotFoundByID) {
		t.Fatalf("expected ErrVideoNotFoundByID, got %v", err)
	}

	// Telemetry: two workers on the same job/range.
	secondWorker := worker
	secondWorker.ID = 50
	secondWorker.Config.Name = "telemetry-worker"
	if err := p.AddWorker(ctx, secondWorker); err != nil {
		t.Fatalf("AddWorker second: %v", err)
	}
	if err := p.AddMetrics(ctx, domain.Metrics{
		JobID: job.ID, RangeID: cascadeRange.ID, WorkerID: worker.ID,
		CPULoad: 0.5, Requests: 10, FoundMaterials: 8,
	}); err != nil {
		t.Fatalf("AddMetrics: %v", err)
	}
	if err := p.AddMetrics(ctx, domain.Metrics{
		JobID: job.ID, RangeID: cascadeRange.ID, WorkerID: secondWorker.ID,
		CPULoad: 0.25, Requests: 20, FoundMaterials: 16,
	}); err != nil {
		t.Fatalf("AddMetrics second: %v", err)
	}
	metrics, err := p.GetWorkerMetricsByID(ctx, worker.ID)
	if err != nil {
		t.Fatalf("GetWorkerMetricsByID: %v", err)
	}
	if metrics.CPULoad != 0.5 || metrics.Requests != 10 {
		t.Fatalf("unexpected worker metrics: %+v", metrics)
	}
	if workersMetrics, err := p.GetWorkersMetrics(ctx); err != nil || len(workersMetrics) != 2 {
		t.Fatalf("GetWorkersMetrics: %v, len=%d", err, len(workersMetrics))
	}
	if jobsMetrics, err := p.GetJobsMetrics(ctx); err != nil || len(jobsMetrics) != 1 {
		t.Fatalf("GetJobsMetrics: %v, len=%d", err, len(jobsMetrics))
	}
	if metrics, err := p.GetJobMetricsByID(ctx, job.ID); err != nil || metrics.Requests != 30 {
		t.Fatalf("GetJobMetricsByID: %v, metrics=%+v", err, metrics)
	}
	if _, err := p.GetWorkerMetricsByID(ctx, 999); !errors.Is(err, repos.ErrWorkerNotFoundByID) {
		t.Fatalf("expected ErrWorkerNotFoundByID, got %v", err)
	}
	if _, err := p.GetJobMetricsByID(ctx, 999); !errors.Is(err, repos.ErrJobNotFoundByID) {
		t.Fatalf("expected ErrJobNotFoundByID, got %v", err)
	}

	if err := p.DeleteWorkerByID(ctx, secondWorker.ID); err != nil {
		t.Fatalf("DeleteWorkerByID second: %v", err)
	}

	// Stop clears workers; delete cascades ranges and videos.
	job, err = p.StopJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("StopJob: %v", err)
	}
	if job.Status != domain.JobStatusStopped || !job.Paused || len(job.Workers) != 0 {
		t.Fatalf("unexpected stopped job: %+v", job)
	}
	if _, err := p.DeleteJob(ctx, job.ID); err != nil {
		t.Fatalf("DeleteJob: %v", err)
	}
	if _, err := p.GetJob(ctx, job.ID); !errors.Is(err, repos.ErrJobNotFoundByID) {
		t.Fatalf("expected ErrJobNotFoundByID, got %v", err)
	}
	if _, err := p.GetRangeByID(ctx, cascadeRange.ID); !errors.Is(err, repos.ErrRangeNotFoundByID) {
		t.Fatalf("expected ErrRangeNotFoundByID, got %v", err)
	}
	if _, err := p.GetVideoByID(ctx, video.ID); !errors.Is(err, repos.ErrVideoNotFoundByID) {
		t.Fatalf("expected ErrVideoNotFoundByID, got %v", err)
	}
	if err := p.DeleteRangesByJobID(ctx, job.ID); err != nil {
		t.Fatalf("DeleteRangesByJobID missing job: %v", err)
	}

	if err := p.DeleteUserByID(ctx, user.ID); err != nil {
		t.Fatalf("DeleteUserByID: %v", err)
	}
	if _, err := p.GetUserByID(ctx, user.ID); !errors.Is(err, repos.ErrUserNotFoundByID) {
		t.Fatalf("expected ErrUserNotFoundByID, got %v", err)
	}
}
