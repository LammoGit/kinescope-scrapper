package domain

import "time"

// Event is a single log entry emitted by the coordinator or a worker
type Event struct {
	Time       time.Time `json:"time" form:"time" binding:"required"`
	Level      string    `json:"level" form:"level" binding:"required,oneof=Error Warning Info Debug"`
	Component  string    `json:"component" form:"component" binding:"required"`
	JobID      int       `json:"job_id" form:"job_id" binding:"min=0"`
	WorkerID   int       `json:"worker_id" form:"worker_id" binding:"min=0"`
	RangeID    int       `json:"range_id" form:"range_id" binding:"min=0"`
	ResourceID int       `json:"resource_id" form:"resource_id" binding:"min=0"`
	Operation  string    `json:"operation_type" form:"operation_type"`
	Result     string    `json:"result" form:"result"`
	ErrorCode  int       `json:"error_code" form:"error_code"`
	Message    string    `json:"message" form:"message"`
}

// EventLevel is the severity of an event
const (
	EventLevelError   string = "Error"
	EventLevelWarning string = "Warning"
	EventLevelInfo    string = "Info"
	EventLevelDebug   string = "Debug"
)
