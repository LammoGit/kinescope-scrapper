//go:build postgres && !sqlite

package storage

import (
	"context"
	"database/sql"

	"coordinator/internal/domain"
)

const metricsColumns = `
	job_id, range_id, worker_id, cpu_load, memory_load, disk_load,
	network_load, free_space_bytes, opened_files, active_connections,
	worker_uptime, identifiers_per_second, requests, found_materials,
	active_downloads, bytes_per_second, avg_check_speed, avg_download_speed,
	download_queue_size, repeated_attempts, final_errors, oldest_in_queue`

func scanMetrics(s rowScanner) (domain.Metrics, error) {
	var m domain.Metrics
	err := s.Scan(
		&m.JobID, &m.RangeID, &m.WorkerID, &m.CPULoad, &m.MemoryLoad,
		&m.DiskLoad, &m.NetworkLoad, &m.FreeSpaceBytes, &m.OpenedFiles,
		&m.ActiveConnections, &m.WorkerUptime, &m.IdentifiersPerSecond,
		&m.Requests, &m.FoundMaterials, &m.ActiveDownloads, &m.BytesPerSecond,
		&m.AvgCheckSpeed, &m.AvgDownloadSpeed, &m.DownloadQueueSize,
		&m.RepeatedAttempts, &m.FinalErrors, &m.OldestInQueue,
	)
	return m, err
}

func (p *Postgres) AddMetrics(ctx context.Context, metrics domain.Metrics) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO telemetry (
			job_id, range_id, worker_id, cpu_load, memory_load, disk_load,
			network_load, free_space_bytes, opened_files, active_connections,
			worker_uptime, identifiers_per_second, requests, found_materials,
			active_downloads, bytes_per_second, avg_check_speed, avg_download_speed,
			download_queue_size, repeated_attempts, final_errors, oldest_in_queue
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22
		)
		ON CONFLICT (job_id, range_id, worker_id) DO UPDATE SET
			cpu_load = EXCLUDED.cpu_load,
			memory_load = EXCLUDED.memory_load,
			disk_load = EXCLUDED.disk_load,
			network_load = EXCLUDED.network_load,
			free_space_bytes = EXCLUDED.free_space_bytes,
			opened_files = EXCLUDED.opened_files,
			active_connections = EXCLUDED.active_connections,
			worker_uptime = EXCLUDED.worker_uptime,
			identifiers_per_second = EXCLUDED.identifiers_per_second,
			requests = EXCLUDED.requests,
			found_materials = EXCLUDED.found_materials,
			active_downloads = EXCLUDED.active_downloads,
			bytes_per_second = EXCLUDED.bytes_per_second,
			avg_check_speed = EXCLUDED.avg_check_speed,
			avg_download_speed = EXCLUDED.avg_download_speed,
			download_queue_size = EXCLUDED.download_queue_size,
			repeated_attempts = EXCLUDED.repeated_attempts,
			final_errors = EXCLUDED.final_errors,
			oldest_in_queue = EXCLUDED.oldest_in_queue,
			updated_at = now()`,
		metrics.JobID, metrics.RangeID, metrics.WorkerID, metrics.CPULoad,
		metrics.MemoryLoad, metrics.DiskLoad, metrics.NetworkLoad,
		metrics.FreeSpaceBytes, metrics.OpenedFiles, metrics.ActiveConnections,
		metrics.WorkerUptime, metrics.IdentifiersPerSecond, metrics.Requests,
		metrics.FoundMaterials, metrics.ActiveDownloads, metrics.BytesPerSecond,
		metrics.AvgCheckSpeed, metrics.AvgDownloadSpeed, metrics.DownloadQueueSize,
		metrics.RepeatedAttempts, metrics.FinalErrors, metrics.OldestInQueue,
	)
	return err
}

func (p *Postgres) GetWorkersMetrics(ctx context.Context) ([]domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := p.db.QueryContext(ctx,
		`SELECT `+metricsColumns+` FROM telemetry ORDER BY worker_id, job_id, range_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return aggregateBy(rows, func(m domain.Metrics) int { return m.WorkerID })
}

func (p *Postgres) GetJobsMetrics(ctx context.Context) ([]domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := p.db.QueryContext(ctx,
		`SELECT `+metricsColumns+` FROM telemetry ORDER BY job_id, worker_id, range_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return aggregateBy(rows, func(m domain.Metrics) int { return m.JobID })
}

func (p *Postgres) GetWorkerMetricsByID(ctx context.Context, id int) (domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return domain.Metrics{}, err
	}

	metrics, err := p.metricsByID(ctx, "worker_id", id)
	if err != nil {
		return domain.Metrics{}, err
	}
	if len(metrics) == 0 {
		return domain.Metrics{}, errWorkerNotFound
	}
	return domain.AverageMetrics(metrics...), nil
}

func (p *Postgres) GetJobMetricsByID(ctx context.Context, id int) (domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return domain.Metrics{}, err
	}

	metrics, err := p.metricsByID(ctx, "job_id", id)
	if err != nil {
		return domain.Metrics{}, err
	}
	if len(metrics) == 0 {
		return domain.Metrics{}, errJobNotFound
	}
	return domain.AverageMetrics(metrics...), nil
}

func (p *Postgres) metricsByID(ctx context.Context, column string, id int) ([]domain.Metrics, error) {
	// column is a private constant chosen by the callers above.
	rows, err := p.db.QueryContext(ctx,
		`SELECT `+metricsColumns+` FROM telemetry WHERE `+column+` = $1 ORDER BY job_id, range_id, worker_id`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := []domain.Metrics{}
	for rows.Next() {
		m, err := scanMetrics(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func aggregateBy(rows *sql.Rows, key func(domain.Metrics) int) ([]domain.Metrics, error) {
	result := []domain.Metrics{}
	currentKey := -1
	var bucket []domain.Metrics

	flush := func() {
		if len(bucket) > 0 {
			result = append(result, domain.AverageMetrics(bucket...))
			bucket = bucket[:0]
		}
	}

	for rows.Next() {
		m, err := scanMetrics(rows)
		if err != nil {
			return nil, err
		}

		k := key(m)
		if currentKey != -1 && k != currentKey {
			flush()
		}
		currentKey = k
		bucket = append(bucket, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	flush()

	return result, nil
}
