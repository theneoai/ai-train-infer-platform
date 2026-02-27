import { useState, useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
} from 'recharts'
import {
  Cpu,
  Database,
  Server,
  Activity,
  Plus,
  Upload,
  Clock,
  TrendingUp,
  TrendingDown,
  User,
  Play,
  CheckCircle,
  XCircle,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import { useAuthStore } from '@/stores/auth'
import type { GPUInfo, Activity as ActivityType, TrainingJob, JobStatus } from '@/types'
import { formatDate, formatBytes } from '@/lib/utils'

// Status badge colors
const statusColors: Record<JobStatus, 'default' | 'secondary' | 'destructive' | 'success' | 'warning' | 'info'> = {
  pending: 'secondary',
  running: 'info',
  completed: 'success',
  failed: 'destructive',
  cancelled: 'warning',
}

interface DashboardStats {
  trainingJobs: number
  datasets: number
  inferenceServices: number
  activeJobs: number
}

// Mock data generators
const generateGPUData = (): GPUInfo[] => [
  { id: 'gpu-0', name: 'NVIDIA A100', utilization: 78, memoryUsed: 32 * 1024 ** 3, memoryTotal: 40 * 1024 ** 3, temperature: 72, powerDraw: 280, powerLimit: 400 },
  { id: 'gpu-1', name: 'NVIDIA A100', utilization: 65, memoryUsed: 28 * 1024 ** 3, memoryTotal: 40 * 1024 ** 3, temperature: 68, powerDraw: 250, powerLimit: 400 },
  { id: 'gpu-2', name: 'NVIDIA A100', utilization: 92, memoryUsed: 38 * 1024 ** 3, memoryTotal: 40 * 1024 ** 3, temperature: 78, powerDraw: 350, powerLimit: 400 },
  { id: 'gpu-3', name: 'NVIDIA A100', utilization: 45, memoryUsed: 16 * 1024 ** 3, memoryTotal: 40 * 1024 ** 3, temperature: 58, powerDraw: 180, powerLimit: 400 },
]

const generateUtilizationHistory = () => {
  const data = []
  const now = new Date()
  for (let i = 23; i >= 0; i--) {
    const time = new Date(now.getTime() - i * 60 * 60 * 1000)
    data.push({
      time: time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
      gpu0: Math.floor(Math.random() * 40) + 50,
      gpu1: Math.floor(Math.random() * 40) + 40,
      gpu2: Math.floor(Math.random() * 40) + 60,
      gpu3: Math.floor(Math.random() * 40) + 30,
    })
  }
  return data
}

const generateRecentActivities = (): ActivityType[] => [
  { id: '1', type: 'job_created', title: 'Training Job Created', description: 'ResNet-50 training job started', timestamp: new Date(Date.now() - 1000 * 60 * 5).toISOString(), user: 'admin' },
  { id: '2', type: 'job_completed', title: 'Training Job Completed', description: 'BERT fine-tuning completed successfully', timestamp: new Date(Date.now() - 1000 * 60 * 30).toISOString(), user: 'admin' },
  { id: '3', type: 'service_deployed', title: 'Inference Service Deployed', description: 'LLaMA-2-7B endpoint is now live', timestamp: new Date(Date.now() - 1000 * 60 * 60).toISOString(), user: 'admin' },
  { id: '4', type: 'experiment_created', title: 'Experiment Created', description: 'Hyperparameter search experiment #42', timestamp: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(), user: 'admin' },
  { id: '5', type: 'job_created', title: 'Training Job Created', description: 'GPT-2 fine-tuning job queued', timestamp: new Date(Date.now() - 1000 * 60 * 60 * 3).toISOString(), user: 'admin' },
]

const generateRecentJobs = (): TrainingJob[] => [
  { id: '1', name: 'ResNet-50 ImageNet', status: 'running', progress: 65, createdAt: '2024-01-15T08:00:00Z', updatedAt: '2024-01-15T10:30:00Z', gpuCount: 4, gpuUsage: 78, memoryUsage: 32 * 1024 ** 3, duration: 3600 * 2.5, modelType: 'ResNet', dataset: 'ImageNet', priority: 'high' },
  { id: '2', name: 'BERT Fine-tuning', status: 'completed', progress: 100, createdAt: '2024-01-14T06:00:00Z', updatedAt: '2024-01-14T14:30:00Z', gpuCount: 2, gpuUsage: 0, memoryUsage: 16 * 1024 ** 3, duration: 3600 * 8.5, modelType: 'BERT', dataset: 'GLUE', priority: 'normal' },
  { id: '3', name: 'GPT-2 Pretraining', status: 'pending', progress: 0, createdAt: '2024-01-15T11:00:00Z', updatedAt: '2024-01-15T11:00:00Z', gpuCount: 8, gpuUsage: 0, memoryUsage: 0, duration: 0, modelType: 'GPT', dataset: 'OpenWebText', priority: 'high' },
]

export function Dashboard() {
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const [loading, setLoading] = useState(true)
  const [gpuData, setGpuData] = useState<GPUInfo[]>([])
  const [utilizationHistory, setUtilizationHistory] = useState<any[]>([])
  const [activities, setActivities] = useState<ActivityType[]>([])
  const [recentJobs, setRecentJobs] = useState<TrainingJob[]>([])
  const [stats, setStats] = useState<DashboardStats>({
    trainingJobs: 0,
    datasets: 0,
    inferenceServices: 0,
    activeJobs: 0,
  })

  useEffect(() => {
    // Simulate loading data
    const timer = setTimeout(() => {
      setGpuData(generateGPUData())
      setUtilizationHistory(generateUtilizationHistory())
      setActivities(generateRecentActivities())
      setRecentJobs(generateRecentJobs())
      setStats({
        trainingJobs: 12,
        datasets: 8,
        inferenceServices: 5,
        activeJobs: 4,
      })
      setLoading(false)
    }, 1000)

    return () => clearTimeout(timer)
  }, [])

  if (loading) {
    return (
      <div className="flex h-[calc(100vh-200px)] items-center justify-center">
        <Spinner size="lg" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Welcome Section */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
            <User className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              欢迎回来, {user?.username || '用户'}!
            </h1>
            <p className="text-muted-foreground">这里是您的 AI 训练和推理平台概览</p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button onClick={() => navigate({ to: '/training' })}>
            <Plus className="mr-2 h-4 w-4" />
            新建训练任务
          </Button>
          <Button variant="outline" onClick={() => navigate({ to: '/datasets' })}>
            <Upload className="mr-2 h-4 w-4" />
            上传数据集
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">训练任务</CardTitle>
            <Cpu className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.trainingJobs}</div>
            <p className="text-xs text-muted-foreground">
              <TrendingUp className="mr-1 inline h-3 w-3 text-green-500" />
              <span className="text-green-500">+3</span> 较上周
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">数据集</CardTitle>
            <Database className="h-4 w-4 text-purple-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.datasets}</div>
            <p className="text-xs text-muted-foreground">
              <TrendingUp className="mr-1 inline h-3 w-3 text-green-500" />
              <span className="text-green-500">+2</span> 较上周
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">推理服务</CardTitle>
            <Server className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.inferenceServices}</div>
            <p className="text-xs text-muted-foreground">
              <span className="text-muted-foreground">0</span> 较上周
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">GPU 使用率</CardTitle>
            <Activity className="h-4 w-4 text-orange-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Math.round(gpuData.reduce((acc, gpu) => acc + gpu.utilization, 0) / gpuData.length)}%
            </div>
            <p className="text-xs text-muted-foreground">
              <TrendingDown className="mr-1 inline h-3 w-3 text-red-500" />
              <span className="text-red-500">-5%</span> 较昨日
            </p>
          </CardContent>
        </Card>
      </div>

      {/* GPU Utilization Chart */}
      <Card>
        <CardHeader>
          <CardTitle>GPU 使用率趋势 (24h)</CardTitle>
          <CardDescription>实时监控所有 GPU 节点的使用情况</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="h-[300px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={utilizationHistory}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="time" className="text-xs" />
                <YAxis className="text-xs" domain={[0, 100]} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'hsl(var(--card))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '6px',
                  }}
                />
                <Area type="monotone" dataKey="gpu0" stackId="1" stroke="#3b82f6" fill="#3b82f6" fillOpacity={0.3} name="GPU 0" />
                <Area type="monotone" dataKey="gpu1" stackId="1" stroke="#10b981" fill="#10b981" fillOpacity={0.3} name="GPU 1" />
                <Area type="monotone" dataKey="gpu2" stackId="1" stroke="#f59e0b" fill="#f59e0b" fillOpacity={0.3} name="GPU 2" />
                <Area type="monotone" dataKey="gpu3" stackId="1" stroke="#8b5cf6" fill="#8b5cf6" fillOpacity={0.3} name="GPU 3" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>

      {/* GPU Status Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {gpuData.map((gpu, index) => (
          <Card key={gpu.id}>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-medium">{gpu.name} #{index}</CardTitle>
                <Badge variant={gpu.utilization > 80 ? 'destructive' : gpu.utilization > 50 ? 'warning' : 'success'}>
                  {gpu.utilization}%
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-2">
              <div className="space-y-1">
                <div className="flex justify-between text-xs">
                  <span className="text-muted-foreground">显存</span>
                  <span>{formatBytes(gpu.memoryUsed)} / {formatBytes(gpu.memoryTotal)}</span>
                </div>
                <div className="h-2 rounded-full bg-muted">
                  <div
                    className="h-2 rounded-full bg-primary transition-all"
                    style={{ width: `${(gpu.memoryUsed / gpu.memoryTotal) * 100}%` }}
                  />
                </div>
              </div>
              <div className="flex justify-between text-xs">
                <span className="text-muted-foreground">温度</span>
                <span className={gpu.temperature > 75 ? 'text-red-500' : gpu.temperature > 65 ? 'text-yellow-500' : 'text-green-500'}>
                  {gpu.temperature}°C
                </span>
              </div>
              <div className="flex justify-between text-xs">
                <span className="text-muted-foreground">功耗</span>
                <span>{gpu.powerDraw}W / {gpu.powerLimit}W</span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Bottom Section */}
      <div className="grid gap-4 lg:grid-cols-3">
        {/* Recent Training Jobs */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>最近训练任务</CardTitle>
                <CardDescription>最新的训练任务状态</CardDescription>
              </div>
              <Button variant="outline" size="sm" onClick={() => navigate({ to: '/training' })}>
                查看全部
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {recentJobs.map((job) => (
                <div
                  key={job.id}
                  className="flex items-center gap-4 rounded-lg border p-3 hover:bg-muted/50 cursor-pointer transition-colors"
                  onClick={() => navigate({ to: '/training/$jobId', params: { jobId: job.id } })}
                >
                  <div className="mt-1 rounded-full bg-muted p-2">
                    {job.status === 'running' && <Play className="h-4 w-4 text-blue-500" />}
                    {job.status === 'completed' && <CheckCircle className="h-4 w-4 text-green-500" />}
                    {job.status === 'pending' && <Clock className="h-4 w-4 text-yellow-500" />}
                    {job.status === 'failed' && <XCircle className="h-4 w-4 text-red-500" />}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium truncate">{job.name}</p>
                      <Badge variant={statusColors[job.status]} className="text-xs">
                        {job.status === 'pending' && '待处理'}
                        {job.status === 'running' && '运行中'}
                        {job.status === 'completed' && '已完成'}
                        {job.status === 'failed' && '失败'}
                        {job.status === 'cancelled' && '已取消'}
                      </Badge>
                    </div>
                    <p className="text-xs text-muted-foreground">{job.modelType} · {job.dataset}</p>
                  </div>
                  <div className="text-right">
                    <div className="text-sm font-medium">{job.progress}%</div>
                    <div className="h-1.5 w-16 rounded-full bg-muted mt-1">
                      <div
                        className="h-1.5 rounded-full bg-primary transition-all"
                        style={{ width: `${job.progress}%` }}
                      />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Recent Activity */}
        <Card>
          <CardHeader>
            <CardTitle>最近活动</CardTitle>
            <CardDescription>平台最新动态</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {activities.map((activity) => (
                <div key={activity.id} className="flex items-start gap-3">
                  <div className="mt-1 rounded-full bg-muted p-1.5">
                    {activity.type === 'job_created' && <Cpu className="h-3 w-3 text-blue-500" />}
                    {activity.type === 'job_completed' && <CheckCircle className="h-3 w-3 text-green-500" />}
                    {activity.type === 'service_deployed' && <Server className="h-3 w-3 text-purple-500" />}
                    {activity.type === 'experiment_created' && <Activity className="h-3 w-3 text-orange-500" />}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{activity.title}</p>
                    <p className="text-xs text-muted-foreground truncate">{activity.description}</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      {formatDate(activity.timestamp)}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Quick Actions */}
      <Card>
        <CardHeader>
          <CardTitle>快速操作</CardTitle>
          <CardDescription>常用任务快捷入口</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Button className="justify-start" variant="outline" onClick={() => navigate({ to: '/training' })}>
              <Plus className="mr-2 h-4 w-4" />
              新建训练任务
            </Button>
            <Button className="justify-start" variant="outline" onClick={() => navigate({ to: '/datasets' })}>
              <Upload className="mr-2 h-4 w-4" />
              上传数据集
            </Button>
            <Button className="justify-start" variant="outline" onClick={() => navigate({ to: '/inference' })}>
              <Server className="mr-2 h-4 w-4" />
              部署推理服务
            </Button>
            <Button className="justify-start" variant="outline" onClick={() => navigate({ to: '/experiments' as '/inference' })}>
              <Activity className="mr-2 h-4 w-4" />
              查看实验
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
