import { useState, useEffect } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
  BarChart,
  Bar,
} from 'recharts'
import {
  Cpu,
  Server,
  FlaskConical,
  Activity,
  Plus,
  Play,
  Settings,
  TrendingUp,
  TrendingDown,
  Clock,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import type { GPUInfo, Activity as ActivityType } from '@/types'
import { formatDate, formatBytes, formatDuration } from '@/lib/utils'

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

const stats = [
  { title: 'Active Training Jobs', value: '12', change: '+3', changeType: 'positive', icon: Cpu, color: 'text-blue-500' },
  { title: 'Inference Services', value: '8', change: '+1', changeType: 'positive', icon: Server, color: 'text-green-500' },
  { title: 'Running Simulations', value: '5', change: '0', changeType: 'neutral', icon: FlaskConical, color: 'text-purple-500' },
  { title: 'GPU Utilization', value: '78%', change: '+5%', changeType: 'positive', icon: Activity, color: 'text-orange-500' },
]

export function Dashboard() {
  const [loading, setLoading] = useState(true)
  const [gpuData, setGpuData] = useState<GPUInfo[]>([])
  const [utilizationHistory, setUtilizationHistory] = useState<any[]>([])
  const [activities, setActivities] = useState<ActivityType[]>([])

  useEffect(() => {
    // Simulate loading data
    const timer = setTimeout(() => {
      setGpuData(generateGPUData())
      setUtilizationHistory(generateUtilizationHistory())
      setActivities(generateRecentActivities())
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
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
          <p className="text-muted-foreground">AI Train-Infer-Sim Platform Overview</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm">
            <Settings className="mr-2 h-4 w-4" />
            Settings
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.title}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{stat.title}</CardTitle>
              <stat.icon className={`h-4 w-4 ${stat.color}`} />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stat.value}</div>
              <p className="text-xs text-muted-foreground">
                {stat.changeType === 'positive' && <TrendingUp className="mr-1 inline h-3 w-3 text-green-500" />}
                {stat.changeType === 'negative' && <TrendingDown className="mr-1 inline h-3 w-3 text-red-500" />}
                <span className={stat.changeType === 'positive' ? 'text-green-500' : stat.changeType === 'negative' ? 'text-red-500' : ''}>
                  {stat.change}
                </span>
                {' '}from last hour
              </p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* GPU Utilization Chart */}
      <Card>
        <CardHeader>
          <CardTitle>GPU Utilization (24h)</CardTitle>
          <CardDescription>Real-time GPU usage across all nodes</CardDescription>
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
                  <span className="text-muted-foreground">Memory</span>
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
                <span className="text-muted-foreground">Temperature</span>
                <span className={gpu.temperature > 75 ? 'text-red-500' : gpu.temperature > 65 ? 'text-yellow-500' : 'text-green-500'}>
                  {gpu.temperature}°C
                </span>
              </div>
              <div className="flex justify-between text-xs">
                <span className="text-muted-foreground">Power</span>
                <span>{gpu.powerDraw}W / {gpu.powerLimit}W</span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Bottom Section: Activity & Quick Actions */}
      <div className="grid gap-4 lg:grid-cols-3">
        {/* Recent Activity */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
            <CardDescription>Latest platform activities and events</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {activities.map((activity) => (
                <div key={activity.id} className="flex items-start gap-4">
                  <div className="mt-1 rounded-full bg-muted p-2">
                    {activity.type === 'job_created' && <Cpu className="h-4 w-4 text-blue-500" />}
                    {activity.type === 'job_completed' && <Play className="h-4 w-4 text-green-500" />}
                    {activity.type === 'service_deployed' && <Server className="h-4 w-4 text-purple-500" />}
                    {activity.type === 'experiment_created' && <FlaskConical className="h-4 w-4 text-orange-500" />}
                  </div>
                  <div className="flex-1 space-y-1">
                    <p className="text-sm font-medium">{activity.title}</p>
                    <p className="text-sm text-muted-foreground">{activity.description}</p>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Clock className="h-3 w-3" />
                      {formatDate(activity.timestamp)}
                    </div>
                  </div>                
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Quick Actions */}
        <Card>
          <CardHeader>
            <CardTitle>Quick Actions</CardTitle>
            <CardDescription>Common tasks and operations</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <Button className="w-full justify-start" variant="outline">
              <Plus className="mr-2 h-4 w-4" />
              New Training Job
            </Button>
            <Button className="w-full justify-start" variant="outline">
              <Server className="mr-2 h-4 w-4" />
              Deploy Inference Service
            </Button>
            <Button className="w-full justify-start" variant="outline">
              <FlaskConical className="mr-2 h-4 w-4" />
              Start Simulation
            </Button>
            <Button className="w-full justify-start" variant="outline">
              <Activity className="mr-2 h-4 w-4" />
              View GPU Metrics
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
