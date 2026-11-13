package domain

// Metrics structure holding telemetry data sent by a worker
type Metrics struct {
	RangeID              int     `json:"range_id" form:"range_id" binding:"required,min=0"`                             // ID of the range metrics belong to
	JobID                int     `json:"job_id" form:"job_id" binding:"required,min=0"`                                 // ID of the job metrics belong to
	WorkerID             int     `json:"worker_id" form:"worker_id" binding:"required,min=0"`                           // ID of the worker metrics belong to
	CPULoad              float64 `json:"cpu_load" form:"cpu_load" binding:"required,min=0,max=1"`                       // CPU load of the worker in percent
	MemoryLoad           float64 `json:"memory_load" form:"memory_load" binding:"required,min=0,max=1"`                 // RAM & SWAP load of the worker in percent
	DiskLoad             float64 `json:"disk_load" form:"disk_load" binding:"required,min=0,max=1"`                     // Disk load of the worker in percent
	NetworkLoad          float64 `json:"network_load" form:"network_load" binding:"required,min=0,max=1"`               // Network load of the worker in percent
	FreeSpaceBytes       int     `json:"free_space_bytes" form:"free_space_bytes" binding:"required,min=0"`             // Number of free bytes left for the worker
	OpenedFiles          int     `json:"opened_files" form:"opened_files" binding:"required,min=0"`                     // Number of files opened by the worker
	ActiveConnections    int     `json:"active_connections" form:"active_connections" binding:"required,min=0"`         // Number of connections opened by the worker
	WorkerUptime         float64 `json:"worker_uptime" form:"worker_uptime" binding:"required,min=0"`                   // Uptime of the worker in seconds
	IdentifiersPerSecond float64 `json:"identifiers_per_second" form:"identifiers_per_second" binding:"required,min=0"` // Number of identifiers being processed per second by the worker
	Requests             int     `json:"requests" form:"requests" binding:"required,min=0"`                             // Number of requests sent by the worker for this job
	FoundMaterials       int     `json:"found_materials" form:"found_materials" binding:"required,min=0"`               // Number of materials found by the worker for this job
	ActiveDownloads      int     `json:"active_downloads" form:"active_downloads" binding:"required,min=0"`             // Number of active downloads of the worker
	BytesPerSecond       int     `json:"bytes_per_second" form:"bytes_per_second" binding:"required,min=0"`             // Number of bytes being downloaded by the worker per second
	AvgCheckSpeed        float64 `json:"avg_check_speed" form:"avg_check_speed" binding:"required,min=0"`               // Average speed of checking an identifier
	AvgDownloadSpeed     float64 `json:"avg_download_speed" form:"avg_download_speed" binding:"required,min=0"`         // Average speed of downloading an identifier
	DownloadQueueSize    int     `json:"download_queue_size" form:"download_queue_size" binding:"required,min=0"`       // Current number of downloads in the download queue
	RepeatedAttempts     int     `json:"repeated_attempts" form:"repeated_attempts" binding:"required,min=0"`           // Number of repeated attempts of the worker for this job
	FinalErrors          int     `json:"final_errors" form:"final_errors" binding:"required,min=0"`                     // Number of final errors of the worker for this job
	OldestInQueue        int     `json:"oldest_in_queue" form:"oldest_in_queue" binding:"required,min=0"`               // Age of the oldest download in the download queue of the worker
}

// AverageMetrics calculates the aggregate of multiple metrics snapshots
// Identifiers that differ between snapshots are set to -1
func AverageMetrics(metrics ...Metrics) (nm Metrics) {
	if len(metrics) == 0 {
		return
	}
	if len(metrics) == 1 {
		return metrics[0]
	}

	nm.RangeID = metrics[0].RangeID
	nm.JobID = metrics[0].JobID
	nm.WorkerID = metrics[0].WorkerID
	for _, m := range metrics {
		if nm.RangeID != m.RangeID {
			nm.RangeID = 0
		}
		if nm.JobID != m.JobID {
			nm.JobID = 0
		}
		if nm.WorkerID != m.WorkerID {
			nm.WorkerID = 0
		}
		nm.CPULoad += m.CPULoad
		nm.MemoryLoad += m.MemoryLoad
		nm.DiskLoad += m.DiskLoad
		nm.NetworkLoad += m.NetworkLoad
		nm.FreeSpaceBytes += m.FreeSpaceBytes
		nm.OpenedFiles += m.OpenedFiles
		nm.ActiveConnections += m.ActiveConnections
		nm.WorkerUptime += m.WorkerUptime
		nm.IdentifiersPerSecond += m.IdentifiersPerSecond
		nm.Requests += m.Requests
		nm.FoundMaterials += m.FoundMaterials
		nm.ActiveDownloads += m.ActiveDownloads
		nm.BytesPerSecond += m.BytesPerSecond
		nm.AvgCheckSpeed += m.AvgCheckSpeed
		nm.AvgDownloadSpeed += m.AvgDownloadSpeed
		nm.DownloadQueueSize += m.DownloadQueueSize
		nm.RepeatedAttempts += m.RepeatedAttempts
		nm.FinalErrors += m.FinalErrors
		nm.OldestInQueue = max(nm.OldestInQueue, m.OldestInQueue)
	}

	// Convert fields from sums to averages of averages
	norm := 1 / float64(len(metrics))
	nm.CPULoad *= norm
	nm.MemoryLoad *= norm
	nm.DiskLoad *= norm
	nm.NetworkLoad *= norm
	nm.WorkerUptime *= norm
	nm.AvgCheckSpeed *= norm
	nm.AvgDownloadSpeed *= norm

	return
}
