import { useState, useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
import {
  Search,
  Filter,
  Plus,
  Square,
  Trash2,
  Clock,
  Cpu,
  ChevronLeft,
  ChevronRight,
  AlertCircle,
  RefreshCw,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import type { TrainingJob, JobStatus } from '@/types'
import { formatDate, formatDuration } from '@/lib/utils'
import { trainingApi } from '@/services/api'

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

interface CreateJobFormData {
  name: string
  dataset_id: string
  gpu_count: number
  config: {
    learning_rate: number
    batch_size: number
    epochs: number
  }
}

export function TrainingList() {
  const navigate = useNavigate()
  const [jobs, setJobs] = useState<TrainingJob[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [datasets] = useState<Array<{ id: string; name: string }>>([
    { id: '1', name: 'ImageNet-1K' },
    { id: '2', name: 'COCO 2017' },
    { id: '3', name: 'GLUE Benchmark' },
    { id: '4', name: 'Custom Dataset' },
  ])

  // Pagination
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(10)

  // Form state
  const [formData, setFormData] = useState<CreateJobFormData>({
    name: '',
    dataset_id: '',
    gpu_count: 1,
    config: {
      learning_rate: 0.001,
      batch_size: 32,
      epochs: 10,
    },
  })

  // Fetch jobs
  const fetchJobs = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await trainingApi.list() as unknown as { jobs: TrainingJob[]; total: number }
      if (response && response.jobs) {
        setJobs(response.jobs)
      } else {
        setJobs(Array.isArray(response) ? response as TrainingJob[] : [])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取训练任务失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchJobs()
  }, [])

  // Filter jobs
  const filteredJobs = jobs.filter((job) => {
    const matchesSearch = job.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         job.modelType?.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         job.dataset?.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesStatus = statusFilter === 'all' || job.status === statusFilter
    return matchesSearch && matchesStatus
  })

  // Pagination
  const paginatedJobs = filteredJobs.slice((currentPage - 1) * pageSize, currentPage * pageSize)
  const totalPages = Math.ceil(filteredJobs.length / pageSize)

  const handleCreateJob = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsSubmitting(true)
    try {
      await trainingApi.create({
        name: formData.name,
        dataset_id: formData.dataset_id,
        gpu_count: formData.gpu_count,
        config: formData.config,
      })
      setIsCreateDialogOpen(false)
      setFormData({
        name: '',
        dataset_id: '',
        gpu_count: 1,
        config: {
          learning_rate: 0.001,
          batch_size: 32,
          epochs: 10,
        },
      })
      fetchJobs() // Refresh list
    } catch (err) {
      alert(err instanceof Error ? err.message : '创建任务失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleDeleteJob = async (jobId: string) => {
    if (!confirm('确定要删除这个训练任务吗？')) return
    try {
      await trainingApi.delete(jobId)
      fetchJobs() // Refresh list
    } catch (err) {
      alert(err instanceof Error ? err.message : '删除任务失败')
    }
  }

  const handleStopJob = async (jobId: string) => {
    if (!confirm('确定要停止这个训练任务吗？')) return
    try {
      await trainingApi.stop(jobId)
      fetchJobs() // Refresh list
    } catch (err) {
      alert(err instanceof Error ? err.message : '停止任务失败')
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">训练任务</h1>
          <p className="text-muted-foreground">管理您的 AI 训练工作负载</p>
        </div>
        <Button onClick={() => setIsCreateDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          新建训练任务
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col gap-4 sm:flex-row">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="搜索任务名称、模型类型或数据集..."
                className="pl-10"
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value)
                  setCurrentPage(1)
                }}
              />
            </div>
            <div className="flex gap-2">
              <Select value={statusFilter} onValueChange={(v: string) => { setStatusFilter(v); setCurrentPage(1); }}>
                <SelectTrigger className="w-[140px]">
                  <Filter className="mr-2 h-4 w-4" />
                  <SelectValue placeholder="状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部状态</SelectItem>
                  <SelectItem value="pending">待处理</SelectItem>
                  <SelectItem value="running">运行中</SelectItem>
                  <SelectItem value="completed">已完成</SelectItem>
                  <SelectItem value="failed">失败</SelectItem>
                  <SelectItem value="cancelled">已取消</SelectItem>
                </SelectContent>
              </Select>
              <Button variant="outline" size="icon" onClick={fetchJobs} disabled={loading}>
                <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Jobs Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>全部任务</CardTitle>
              <CardDescription>共 {filteredJobs.length} 个任务</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {error && (
            <div className="mb-4 flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
              <AlertCircle className="h-4 w-4" />
              {error}
            </div>
          )}
          
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>任务名称</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>进度</TableHead>
                  <TableHead>GPU</TableHead>
                  <TableHead>运行时长</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-24 text-center">
                      <div className="flex items-center justify-center gap-2">
                        <Spinner size="sm" />
                        加载中...
                      </div>
                    </TableCell>
                  </TableRow>
                ) : paginatedJobs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                      暂无训练任务
                    </TableCell>
                  </TableRow>
                ) : (
                  paginatedJobs.map((job) => (
                    <TableRow
                      key={job.id}
                      className="cursor-pointer"
                      onClick={() => navigate({ to: '/training/$jobId', params: { jobId: job.id } })}
                    >
                      <TableCell>
                        <div className="font-medium">{job.name}</div>
                        <div className="text-xs text-muted-foreground">{job.modelType} · {job.dataset}</div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusColors[job.status]}>
                          {statusLabels[job.status]}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <div className="h-2 w-16 rounded-full bg-muted">
                            <div
                              className="h-2 rounded-full bg-primary transition-all"
                              style={{ width: `${job.progress}%` }}
                            />
                          </div>
                          <span className="text-xs text-muted-foreground w-8">{job.progress}%</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Cpu className="h-3 w-3" />
                          <span className="text-xs">{job.gpuCount}</span>
                          {job.status === 'running' && (
                            <span className="text-xs text-muted-foreground">({job.gpuUsage}%)</span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          <span className="text-xs">{formatDuration(job.duration)}</span>
                        </div>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatDate(job.createdAt)}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
                          {job.status === 'running' && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                              onClick={() => handleStopJob(job.id)}
                              title="停止"
                            >
                              <Square className="h-4 w-4" />
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8"
                            onClick={() => handleDeleteJob(job.id)}
                            title="删除"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="mt-4 flex items-center justify-between">
              <div className="text-sm text-muted-foreground">
                显示 {(currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, filteredJobs.length)} 共 {filteredJobs.length} 个任务
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <span className="text-sm">{currentPage} / {totalPages}</span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Job Dialog */}
      <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>创建训练任务</DialogTitle>
            <DialogDescription>
              配置您的新训练任务。请填写以下详细信息。
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleCreateJob} className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">任务名称</label>
              <Input
                placeholder="例如：ResNet-50 图像分类"
                required
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">选择数据集</label>
              <Select
                required
                value={formData.dataset_id}
                onValueChange={(v: string) => setFormData({ ...formData, dataset_id: v })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择数据集" />
                </SelectTrigger>
                <SelectContent>
                  {datasets.map((ds) => (
                    <SelectItem key={ds.id} value={ds.id}>{ds.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">GPU 数量</label>
                <Select
                  value={formData.gpu_count.toString()}
                  onValueChange={(v: string) => setFormData({ ...formData, gpu_count: parseInt(v) })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择 GPU" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="1">1 GPU</SelectItem>
                    <SelectItem value="2">2 GPUs</SelectItem>
                    <SelectItem value="4">4 GPUs</SelectItem>
                    <SelectItem value="8">8 GPUs</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Batch Size</label>
                <Input
                  type="number"
                  min={1}
                  value={formData.config.batch_size}
                  onChange={(e) => setFormData({
                    ...formData,
                    config: { ...formData.config, batch_size: parseInt(e.target.value) }
                  })}
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">学习率</label>
                <Input
                  type="number"
                  step="0.0001"
                  min={0}
                  value={formData.config.learning_rate}
                  onChange={(e) => setFormData({
                    ...formData,
                    config: { ...formData.config, learning_rate: parseFloat(e.target.value) }
                  })}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">训练轮数</label>
                <Input
                  type="number"
                  min={1}
                  value={formData.config.epochs}
                  onChange={(e) => setFormData({
                    ...formData,
                    config: { ...formData.config, epochs: parseInt(e.target.value) }
                  })}
                />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsCreateDialogOpen(false)}>
                取消
              </Button>
              <Button type="submit" disabled={isSubmitting || !formData.name || !formData.dataset_id}>
                {isSubmitting ? (
                  <>
                    <Spinner className="mr-2" size="sm" />
                    创建中...
                  </>
                ) : '创建任务'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
