package domain

import (
	"context"
	"time"
)

// IEventRepository stores coordinator and worker events
type IEventRepository interface {
	AddEvent(ctx context.Context, event Event) error
	GetEvents(ctx context.Context) ([]Event, error)
	GetEventsByJobID(ctx context.Context, jobID int) ([]Event, error)
	GetEventsByWorkerID(ctx context.Context, workerID int) ([]Event, error)
}

// EventLevel is the severity of an event
const (
	EventLevelError   string = "Error"
	EventLevelWarning string = "Warning"
	EventLevelInfo    string = "Info"
	EventLevelDebug   string = "Debug"
)

// Event is a single log entry emitted by the coordinator or a worker
type Event struct {
	Time       time.Time `json:"time" form:"time" binding:"required"`
	Level      string    `json:"level" form:"level" binding:"required,oneof=Error Warning Info Debug"`
	Component  string    `json:"component" form:"component" binding:"required"`
	JobID      int       `json:"job_id" form:"job_id" binding:"omitempty,min=0"`
	WorkerID   int       `json:"worker_id" form:"worker_id" binding:"omitempty,min=0"`
	RangeID    int       `json:"range_id" form:"range_id" binding:"omitempty,min=0"`
	ResourceID int       `json:"resource_id" form:"resource_id" binding:"omitempty,min=0"`
	Operation  string    `json:"operation_type" form:"operation_type" binding:"required"`
	Result     string    `json:"result" form:"result" binding:"required"`
	ErrorCode  int       `json:"error_code" form:"error_code" binding:"required,min=0"`
	Message    string    `json:"message" form:"message" binding:"required"`
}

// Clone creates a deep-copy of the calling Event
func (e Event) Clone() Event {
	return e
}
