import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate } from '@tanstack/react-router'
import {
  ArrowLeft,
  Square,
  Trash2,
  Clock,
  Cpu,
  HardDrive,
  Activity,
  AlertCircle,
  CheckCircle,
  Play,
  RefreshCw,
  Terminal,
  BarChart3,
  FileText,
  Settings,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { TrainingJob, JobStatus } from '@/types'
import { formatDate, formatDuration, formatBytes } from '@/lib/utils'
import { trainingApi } from '@/services/api'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'

const statusColors: Record<JobStatus, 'default' | 'secondary' | 'destructive' | 'success' | 'warning' | 'info'> = {
  pending: 'secondary',
  running: 'info',
  completed: 'success',
  failed: 'destructive',
  cancelled: 'warning',
}

const statusLabels: Record<JobStatus, string> = {
  pending: '待处理',
  running: '运行中',
  completed: '已完成',
  failed: '失败',
  cancelled: '已取消',
}

// Mock metrics data
const generateMetricsData = () => {
  const data = []
  for (let i = 0; i < 100; i++) {
    data.push({
      step: i,
      loss: 2.5 * Math.exp(-i / 30) + 0.1 + Math.random() * 0.05,
      accuracy: Math.min(0.95, 0.5 + i * 0.004 + Math.random() * 0.02),
      val_loss: 2.5 * Math.exp(-i / 35) + 0.15 + Math.random() * 0.08,
      val_accuracy: Math.min(0.92, 0.48 + i * 0.0038 + Math.random() * 0.025),
    })
  }
  return data
}

export function TrainingDetail() {
  const { jobId } = useParams({ from: '/training/$jobId' })
  const navigate = useNavigate()
  const [job, setJob] = useState<TrainingJob | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [logs, setLogs] = useState<string[]>([])
  const [metricsData, setMetricsData] = useState<any[]>([])
  const [isStopping, setIsStopping] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const eventSourceRef = useRef<EventSource | null>(null)

  // Fetch job details
  const fetchJob = async () => {
    try {
      setError(null)
      const response = await trainingApi.get(jobId) as TrainingJob
      setJob(response)
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取任务详情失败')
    }
  }

  // Initial fetch
  useEffect(() => {
    const init = async () => {
      setLoading(true)
      await fetchJob()
      setMetricsData(generateMetricsData())
      setLoading(false)
    }
    init()
  }, [jobId])

  // Setup SSE for logs
  useEffect(() => {
    if (!jobId || job?.status === 'completed' || job?.status === 'failed' || job?.status === 'cancelled') {
      // For completed jobs, show mock logs
      setLogs([
        '[INFO] Training job initialized',
        '[INFO] Loading dataset...',
        '[INFO] Dataset loaded: 50000 samples',
        '[INFO] Model architecture: ResNet-50',
        '[INFO] Starting training...',
        '[INFO] Epoch 1/10 - Loss: 2.345 - Accuracy: 0.456',
        '[INFO] Epoch 2/10 - Loss: 1.876 - Accuracy: 0.567',
        '[INFO] Epoch 3/10 - Loss: 1.456 - Accuracy: 0.678',
        '[INFO] Saving checkpoint...',
        '[INFO] Checkpoint saved to /checkpoints/model_epoch_3.pth',
        '...',
      ])
      return
    }

    // Connect to SSE endpoint
    const logsUrl = trainingApi.getLogs(jobId)
    const eventSource = new EventSource(logsUrl)
    eventSourceRef.current = eventSource

    eventSource.onmessage = (event) => {
      setLogs((prev) => [...prev, event.data])
    }

    eventSource.onerror = () => {
      eventSource.close()
    }

    return () => {
      eventSource.close()
    }
  }, [jobId, job?.status])

  // Auto-scroll logs to bottom
  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  // Poll job status while running
  useEffect(() => {
    if (job?.status !== 'running') return

    const interval = setInterval(() => {
      fetchJob()
    }, 5000)

    return () => clearInterval(interval)
  }, [job?.status])

  const handleStop = async () => {
    if (!confirm('确定要停止这个训练任务吗？')) return
    setIsStopping(true)
    try {
      await trainingApi.stop(jobId)
      await fetchJob()
    } catch (err) {
      alert(err instanceof Error ? err.message : '停止任务失败')
    } finally {
      setIsStopping(false)
    }
  }

  const handleDelete = async () => {
    if (!confirm('确定要删除这个训练任务吗？此操作不可恢复。')) return
    setIsDeleting(true)
    try {
      await trainingApi.delete(jobId)
      navigate({ to: '/training' })
    } catch (err) {
      alert(err instanceof Error ? err.message : '删除任务失败')
      setIsDeleting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex h-[calc(100vh-200px)] items-center justify-center">
        <Spinner size="lg" />
      </div>
    )
  }

  if (error || !job) {
    return (
      <div className="space-y-4">
        <Button variant="outline" onClick={() => navigate({ to: '/training' })}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          返回列表
        </Button>
        <Card>
          <CardContent className="flex h-[300px] items-center justify-center">
            <div className="text-center">
              <AlertCircle className="mx-auto h-12 w-12 text-destructive" />
              <p className="mt-4 text-lg font-medium">{error || '任务不存在'}</p>
              <Button className="mt-4" variant="outline" onClick={fetchJob}>
                <RefreshCw className="mr-2 h-4 w-4" />
                重试
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="icon" onClick={() => navigate({ to: '/training' })}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold tracking-tight">{job.name}</h1>
              <Badge variant={statusColors[job.status]}>{statusLabels[job.status]}</Badge>
            </div>
            <p className="text-sm text-muted-foreground">ID: {job.id}</p>
          </div>
        </div>
        <div className="flex gap-2">
          {job.status === 'running' && (
            <Button variant="destructive" onClick={handleStop} disabled={isStopping}>
              {isStopping ? (
                <><Spinner className="mr-2" size="sm" />停止中...</>
              ) : (
                <><Square className="mr-2 h-4 w-4" />停止</>
              )}
            </Button>
          )}
          <Button variant="outline" onClick={handleDelete} disabled={isDeleting}>
            {isDeleting ? (
              <><Spinner className="mr-2" size="sm" />删除中...</>
            ) : (
              <><Trash2 className="mr-2 h-4 w-4" />删除</>
            )}
          </Button>
        </div>
      </div>

      {/* Progress Bar */}
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">训练进度</span>
            <span className="text-sm font-medium">{job.progress}%</span>
          </div>
          <div className="mt-2 h-2 rounded-full bg-muted">
            <div
              className="h-2 rounded-full bg-primary transition-all"
              style={{ width: `${job.progress}%` }}
            />
          </div>
        </CardContent>
      </Card>

      {/* Info Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">模型类型</CardTitle>
            <Settings className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-lg font-medium">{job.modelType || '-'}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">数据集</CardTitle>
            <HardDrive className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-lg font-medium truncate">{job.dataset || '-'}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">GPU</CardTitle>
            <Cpu className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-lg font-medium">{job.gpuCount} 个</div>
            {job.status === 'running' && (
              <p className="text-xs text-muted-foreground">使用率: {job.gpuUsage}%</p>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">运行时长</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-lg font-medium">{formatDuration(job.duration)}</div>
          </CardContent>
        </Card>
      </div>

      {/* Tabs: Logs, Metrics, Details */}
      <Tabs defaultValue="logs" className="space-y-4">
        <TabsList>
          <TabsTrigger value="logs">
            <Terminal className="mr-2 h-4 w-4" />
            日志
          </TabsTrigger>
          <TabsTrigger value="metrics">
            <BarChart3 className="mr-2 h-4 w-4" />
            指标
          </TabsTrigger>
          <TabsTrigger value="details">
            <FileText className="mr-2 h-4 w-4" />
            详情
          </TabsTrigger>
        </TabsList>

        {/* Logs Tab */}
        <TabsContent value="logs">
          <Card>
            <CardHeader>
              <CardTitle>训练日志</CardTitle>
              <CardDescription>实时训练日志输出</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="rounded-lg bg-muted p-4 font-mono text-sm">
                <div className="max-h-[500px] overflow-auto space-y-1">
                  {logs.length === 0 ? (
                    <span className="text-muted-foreground">暂无日志...</span>
                  ) : (
                    logs.map((log, index) => (
                      <div key={index} className="break-all">
                        <span className="text-muted-foreground">[{new Date().toLocaleTimeString()}]</span>{' '}
                        {log}
                      </div>
                    ))
                  )}
                  <div ref={logsEndRef} />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Metrics Tab */}
        <TabsContent value="metrics">
          <div className="grid gap-4">
            <Card>
              <CardHeader>
                <CardTitle>损失曲线</CardTitle>
                <CardDescription>训练损失和验证损失变化</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="h-[300px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={metricsData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="step" label={{ value: 'Step', position: 'insideBottom', offset: -5 }} />
                      <YAxis label={{ value: 'Loss', angle: -90, position: 'insideLeft' }} />
                      <Tooltip />
                      <Legend />
                      <Line type="monotone" dataKey="loss" stroke="#3b82f6" name="Training Loss" dot={false} />
                      <Line type="monotone" dataKey="val_loss" stroke="#ef4444" name="Validation Loss" dot={false} strokeDasharray="5 5" />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>准确率曲线</CardTitle>
                <CardDescription>训练准确率和验证准确率变化</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="h-[300px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={metricsData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="step" label={{ value: 'Step', position: 'insideBottom', offset: -5 }} />
                      <YAxis domain={[0, 1]} label={{ value: 'Accuracy', angle: -90, position: 'insideLeft' }} />
                      <Tooltip />
                      <Legend />
                      <Line type="monotone" dataKey="accuracy" stroke="#10b981" name="Training Accuracy" dot={false} />
                      <Line type="monotone" dataKey="val_accuracy" stroke="#f59e0b" name="Validation Accuracy" dot={false} strokeDasharray="5 5" />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Details Tab */}
        <TabsContent value="details">
          <Card>
            <CardHeader>
              <CardTitle>任务详情</CardTitle>
              <CardDescription>训练任务的详细信息</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">任务 ID</label>
                    <p className="text-sm font-mono">{job.id}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">任务名称</label>
                    <p className="text-sm">{job.name}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">状态</label>
                    <p className="text-sm">
                      <Badge variant={statusColors[job.status]}>{statusLabels[job.status]}</Badge>
                    </p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">优先级</label>
                    <p className="text-sm">{job.priority || 'normal'}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">模型类型</label>
                    <p className="text-sm">{job.modelType || '-'}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">数据集</label>
                    <p className="text-sm">{job.dataset || '-'}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">GPU 数量</label>
                    <p className="text-sm">{job.gpuCount} 个</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">显存使用</label>
                    <p className="text-sm">{formatBytes(job.memoryUsage)}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">创建时间</label>
                    <p className="text-sm">{formatDate(job.createdAt)}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">更新时间</label>
                    <p className="text-sm">{formatDate(job.updatedAt)}</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
