export type JobStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
export type ServiceStatus = 'deploying' | 'running' | 'stopped' | 'error'
export type ExperimentStatus = 'active' | 'archived' | 'deleted'

export interface TrainingJob {
  id: string
  name: string
  status: JobStatus
  progress: number
  createdAt: string
  updatedAt: string
  gpuCount: number
  gpuUsage: number
  memoryUsage: number
  duration: number
  modelType: string
  dataset: string
  priority: 'low' | 'normal' | 'high'
}

export interface InferenceService {
  id: string
  name: string
  status: ServiceStatus
  endpoint: string
  modelName: string
  version: string
  createdAt: string
  updatedAt: string
  requestsPerSecond: number
  avgLatency: number
  replicas: number
  gpuUsage: number
}

export interface Experiment {
  id: string
  name: string
  status: ExperimentStatus
  description: string
  createdAt: string
  updatedAt: string
  metrics: Record<string, number>
  tags: string[]
  runCount: number
  bestRunId?: string
}

export interface GPUInfo {
  id: string
  name: string
  utilization: number
  memoryUsed: number
  memoryTotal: number
  temperature: number
  powerDraw: number
  powerLimit: number
}

export interface Activity {
  id: string
  type: 'job_created' | 'job_completed' | 'service_deployed' | 'experiment_created'
  title: string
  description: string
  timestamp: string
  user: string
}
