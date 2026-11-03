package domain

// Metrics structure holding telemetry data sent by a worker
type Metrics struct {
	JobID                int     // ID of the job metrics belong to
	WorkerID             int     // ID of the worker metrics belong to
	CPULoad              float64 `json:"cpu_load"`               // CPU load of the worker in percent
	MemoryLoad           float64 `json:"memory_load"`            // RAM & SWAP load of the worker in percent
	DiskLoad             float64 `json:"disk_load"`              // Disk load of the worker in percent
	NetworkLoad          float64 `json:"network_load"`           // Network load of the worker in percent
	FreeSpaceBytes       int     `json:"free_space_bytes"`       // Number of free bytes left for the worker
	OpenedFiles          int     `json:"opened_files"`           // Number of files openned by the worker
	ActiveConnections    int     `json:"active_connections"`     // Number of connections openned by the worker
	WorkerUptime         float64 `json:"worker_uptime"`          // Uptime of the worker in seconds
	IdentifiersPerSecond float64 `json:"identifiers_per_second"` // Number of identifiers being processed per second by the worker
	Requests             int     `json:"requests"`               // Number of requests sent by the worker for this job
	FoundMaterials       int     `json:"found_materials"`        // Number of materials found by the worker for this job
	ActiveDownloads      int     `json:"active_downloads"`       // Number of active downloads of the worker
	BytesPerSecond       int     `json:"bytes_per_second"`       // Number of bytes being downloaded by the worker per second
	AvgCheckSpeed        float64 `json:"avg_check_speed"`        // Average speed of checking an identifier
	AvgDownloadSpeed     float64 `json:"avg_download_speed"`     // Average speed of downloading an identifier
	DownloadQueueSize    int     `json:"download_queue_size"`    // Current number of downloads in the download queue
	RepeatedAttempts     int     `json:"repeated_attempts"`      // Number of repeated attempts of the worker for this job
	FinalErrors          int     `json:"final_errors"`           // Number of final error of the worker for this job
	OldestInQueue        int     `json:"oldest_in_queue"`        // Age of the oldest download in the download queue of the worker
}

// AverageMetrics calculate average of multiple metrics
func AverageMetrics(metrics ...Metrics) (nm Metrics) {
	if len(metrics) == 0 {
		return
	}
	if len(metrics) == 1 {
		return metrics[0]
	}

	for _, m := range metrics {
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
		nm.ActiveDownloads += m.ActiveDownloads
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
