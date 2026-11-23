export type UserRole = 'viewer' | 'operator' | 'administrator'

export interface PublicUser {
  id: number
  username: string
  email: string
  avatar: string
  role: UserRole
}

export type JobStatus =
  | 'created' | 'validating' | 'running' | 'pausing' | 'paused'
  | 'stopping' | 'stopped' | 'completed' | 'failed'

export interface JobRange { from: number; to: number }

export interface WorkerConfig {
  name: string
  host: string
  ssh_port: number
  ssh_user: string
  ssh_key: string
  threads: number
  storage_path: string
  min_free_space_gb: number
}

export interface JobConfig {
  name: string
  range: JobRange
  request_timeout_seconds: number
  max_tries: number
  global_rate_limit_rps: number
  stop_on_consecutive_missing: number
  servers: WorkerConfig[]
}

export interface Job {
  id: number
  config: JobConfig
  workers: number[]
  status: JobStatus
  paused: boolean
  expiration: string
}

export type WorkerStatus = 'idle' | 'off' | 'upgrading' | 'working'

export interface Worker {
  id: number
  config: WorkerConfig
  status: WorkerStatus
  public_key: string
}

export interface VideoMetadata {
  title: string
  source_url: string
  download_url: string
  found_at: string
  downloaded_at: string
  file_size: number
  mime_type: string
  status: string
}

export interface VideoStorageMetadata {
  bucket_name: string
  object_key: string
  checksum: string
  server_name: string
  local_path: string
}

export interface Video {
  id: number
  job_id: number
  metadata: VideoMetadata
  storage_metadata: VideoStorageMetadata
}

export interface Metrics {
  range_id: number
  job_id: number
  worker_id: number
  cpu_load: number
  memory_load: number
  disk_load: number
  network_load: number
  free_space_bytes: number
  opened_files: number
  active_connections: number
  worker_uptime: number
  identifiers_per_second: number
  requests: number
  found_materials: number
  active_downloads: number
  bytes_per_second: number
  avg_check_speed: number
  avg_download_speed: number
  download_queue_size: number
  repeated_attempts: number
  final_errors: number
  oldest_in_queue: number
}

export type EventLevel = 'Error' | 'Warning' | 'Info' | 'Debug'

export interface AppEvent {
  time: string
  level: EventLevel
  component: string
  job_id: number
  worker_id: number
  range_id: number
  resource_id: number
  operation_type: string
  result: string
  error_code: number
  message: string
}
