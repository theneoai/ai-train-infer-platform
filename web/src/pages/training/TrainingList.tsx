import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import {
  Search,
  Filter,
  Plus,
  MoreHorizontal,
  Play,
  Pause,
  Square,
  Trash2,
  Clock,
  Cpu,
  HardDrive,
  ChevronLeft,
  ChevronRight,
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
  DialogTrigger,
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
import { formatDate, formatBytes, formatDuration } from '@/lib/utils'

// Mock data
const mockJobs: TrainingJob[] = [
  { id: '1', name: 'ResNet-50 ImageNet', status: 'running', progress: 65, createdAt: '2024-01-15T08:00:00Z', updatedAt: '2024-01-15T10:30:00Z', gpuCount: 4, gpuUsage: 78, memoryUsage: 32 * 1024 ** 3, duration: 3600 * 2.5, modelType: 'ResNet', dataset: 'ImageNet', priority: 'high' },
  { id: '2', name: 'BERT Fine-tuning', status: 'completed', progress: 100, createdAt: '2024-01-14T06:00:00Z', updatedAt: '2024-01-14T14:30:00Z', gpuCount: 2, gpuUsage: 0, memoryUsage: 16 * 1024 ** 3, duration: 3600 * 8.5, modelType: 'BERT', dataset: 'GLUE', priority: 'normal' },
  { id: '3', name: 'GPT-2 Pretraining', status: 'pending', progress: 0, createdAt: '2024-01-15T11:00:00Z', updatedAt: '2024-01-15T11:00:00Z', gpuCount: 8, gpuUsage: 0, memoryUsage: 0, duration: 0, modelType: 'GPT', dataset: 'OpenWebText', priority: 'high' },
  { id: '4', name: 'YOLOv8 Detection', status: 'running', progress: 32, createdAt: '2024-01-15T09:30:00Z', updatedAt: '2024-01-15T10:45:00Z', gpuCount: 2, gpuUsage: 92, memoryUsage: 20 * 1024 ** 3, duration: 3600 * 1.25, modelType: 'YOLO', dataset: 'COCO', priority: 'normal' },
  { id: '5', name: 'CLIP Contrastive', status: 'failed', progress: 45, createdAt: '2024-01-14T12:00:00Z', updatedAt: '2024-01-14T15:20:00Z', gpuCount: 4, gpuUsage: 0, memoryUsage: 0, duration: 3600 * 3.33, modelType: 'CLIP', dataset: 'LAION-400M', priority: 'low' },
  { id: '6', name: 'Stable Diffusion Fine-tune', status: 'running', progress: 78, createdAt: '2024-01-15T07:00:00Z', updatedAt: '2024-01-15T10:50:00Z', gpuCount: 4, gpuUsage: 85, memoryUsage: 38 * 1024 ** 3, duration: 3600 * 3.83, modelType: 'Stable Diffusion', dataset: 'Custom', priority: 'high' },
  { id: '7', name: 'T5 Summarization', status: 'cancelled', progress: 23, createdAt: '2024-01-13T10:00:00Z', updatedAt: '2024-01-13T12:30:00Z', gpuCount: 2, gpuUsage: 0, memoryUsage: 0, duration: 3600 * 2.5, modelType: 'T5', dataset: 'CNN/DailyMail', priority: 'low' },
  { id: '8', name: 'ViT Classification', status: 'completed', progress: 100, createdAt: '2024-01-12T08:00:00Z', updatedAt: '2024-01-12T16:00:00Z', gpuCount: 2, gpuUsage: 0, memoryUsage: 0, duration: 3600 * 8, modelType: 'ViT', dataset: 'CIFAR-10', priority: 'normal' },
]

const statusColors: Record<JobStatus, 'default' | 'secondary' | 'destructive' | 'success' | 'warning' | 'info'> = {
  pending: 'secondary',
  running: 'info',
  completed: 'success',
  failed: 'destructive',
  cancelled: 'warning',
}

export function TrainingList() {
  const navigate = useNavigate()
  const [jobs, setJobs] = useState<TrainingJob[]>(mockJobs)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [priorityFilter, setPriorityFilter] = useState<string>('all')
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)

  // Filter jobs
  const filteredJobs = jobs.filter((job) => {
    const matchesSearch = job.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         job.modelType.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesStatus = statusFilter === 'all' || job.status === statusFilter
    const matchesPriority = priorityFilter === 'all' || job.priority === priorityFilter
    return matchesSearch && matchesStatus && matchesPriority
  })

  const handleCreateJob = (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    // Simulate API call
    setTimeout(() => {
      setIsLoading(false)
      setIsCreateDialogOpen(false)
    }, 1000)
  }

  const handleAction = (action: string, jobId: string) => {
    console.log(`${action} job ${jobId}`)
    // Implement action logic
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Training Jobs</h1>
          <p className="text-muted-foreground">Manage your AI training workloads</p>
        </div>
        <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              New Training Job
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Create Training Job</DialogTitle>
              <DialogDescription>
                Configure your new training job. Fill in the details below.
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleCreateJob} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Job Name</label>
                <Input placeholder="e.g., ResNet-50 Training" required />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">Model Type</label>
                  <Select>
                    <SelectTrigger>
                      <SelectValue placeholder="Select model" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="resnet">ResNet</SelectItem>
                      <SelectItem value="bert">BERT</SelectItem>
                      <SelectItem value="gpt">GPT</SelectItem>
                      <SelectItem value="yolo">YOLO</SelectItem>
                      <SelectItem value="custom">Custom</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">Dataset</label>
                  <Input placeholder="Dataset name" required />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">GPU Count</label>
                  <Select defaultValue="1">
                    <SelectTrigger>
                      <SelectValue placeholder="Select GPUs" />
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
                  <label className="text-sm font-medium">Priority</label>
                  <Select defaultValue="normal">
                    <SelectTrigger>
                      <SelectValue placeholder="Select priority" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="low">Low</SelectItem>
                      <SelectItem value="normal">Normal</SelectItem>
                      <SelectItem value="high">High</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setIsCreateDialogOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={isLoading}>
                  {isLoading ? <><Spinner className="mr-2" size="sm" /> Creating...</> : 'Create Job'}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col gap-4 sm:flex-row">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search jobs..."
                className="pl-10"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
            <div className="flex gap-2">
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-[140px]">
                  <Filter className="mr-2 h-4 w-4" />
                  <SelectValue placeholder="Status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Status</SelectItem>
                  <SelectItem value="pending">Pending</SelectItem>
                  <SelectItem value="running">Running</SelectItem>
                  <SelectItem value="completed">Completed</SelectItem>
                  <SelectItem value="failed">Failed</SelectItem>
                  <SelectItem value="cancelled">Cancelled</SelectItem>
                </SelectContent>
              </Select>
              <Select value={priorityFilter} onValueChange={setPriorityFilter}>
                <SelectTrigger className="w-[140px]">
                  <SelectValue placeholder="Priority" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Priority</SelectItem>
                  <SelectItem value="high">High</SelectItem>
                  <SelectItem value="normal">Normal</SelectItem>
                  <SelectItem value="low">Low</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Jobs Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>All Jobs</CardTitle>
              <CardDescription>{filteredJobs.length} jobs found</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Progress</TableHead>
                  <TableHead>GPU</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredJobs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-24 text-center">
                      No jobs found
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredJobs.map((job) => (
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
                          {job.status}
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
                          <span className="text-xs text-muted-foreground">{job.progress}%</span>
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
                          {job.status === 'pending' && (
                            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => handleAction('start', job.id)}>
                              <Play className="h-4 w-4" />
                            </Button>
                          )}
                          {job.status === 'running' && (
                            <>
                              <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => handleAction('pause', job.id)}>
                                <Pause className="h-4 w-4" />
                              </Button>
                              <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => handleAction('stop', job.id)}>
                                <Square className="h-4 w-4" />
                              </Button>
                            </>
                          )}
                          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => handleAction('delete', job.id)}>
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
          <div className="mt-4 flex items-center justify-between">
            <div className="text-sm text-muted-foreground">
              Showing {filteredJobs.length} of {jobs.length} jobs
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" disabled>
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <Button variant="outline" size="sm" disabled>
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
