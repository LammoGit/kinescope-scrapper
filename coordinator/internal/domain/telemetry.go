package domain

type Metrics struct {
	JobID                int
	WorkerID             int
	CPULoad              float64 `json:"cpu_load"`
	MemoryLoad           float64 `json:"memory_load"`
	DiskLoad             float64 `json:"disk_load"`
	NetworkLoad          float64 `json:"network_load"`
	FreeSpaceBytes       int     `json:"free_space_bytes"`
	OpenedFiles          int     `json:"opened_files"`
	ActiveConnections    int     `json:"active_connections"`
	WorkerUptime         float64 `json:"worker_uptime"`
	IdentifiersPerSecond float64 `json:"identifiers_per_second"`
	Requests             int     `json:"requests"`
	FoundMaterials       int     `json:"found_materials"`
	ActiveDownloads      int     `json:"active_downloads"`
	BytesPerSecond       int     `json:"bytes_per_second"`
	AvgCheckSpeed        float64 `json:"avg_check_speed"`
	AvgDownloadSpeed     float64 `json:"avg_download_speed"`
	DownloadQueueSize    int     `json:"download_queue_size"`
	RepeatedAttempts     int     `json:"repeated_attempts"`
	FinalErrors          int     `json:"final_errors"`
	OldestInQueue        int     `json:"oldest_in_queue"`
}

func AverageMetrics(metrics []Metrics) (nm Metrics) {
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
