package domain

import "time"

type Event struct {
	Time       time.Time  `json:"time"`
	Level      EventLevel `json:"level"`
	Component  string     `json:"component"`
	JobID      int        `json:"job_id"`
	WorkerID   int        `json:"worker_id"`
	RangeID    int        `json:"range_id"`
	ResourceID int        `json:"resource_id"`
	Operation  string     `json:"operation_type"`
	Result     string     `json:"result"`
	ErrorCode  int        `json:"error_code"`
	Message    string     `json:"message"`
}

type EventLevel string

const (
	EventLevelError   EventLevel = "Error"
	EventLevelWarning EventLevel = "Warning"
	EventLevelInfo    EventLevel = "Info"
	EventLevelDebug   EventLevel = "Debug"
)
