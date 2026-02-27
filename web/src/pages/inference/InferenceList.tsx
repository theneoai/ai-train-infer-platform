import { useState } from 'react'
import {
  Search,
  Plus,
  ExternalLink,
  Settings,
  Trash2,
  Play,
  Square,
  RefreshCw,
  Copy,
  TrendingUp,
  Clock,
  Cpu,
  Users,
  CheckCircle,
  XCircle,
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
import type { InferenceService, ServiceStatus } from '@/types'
import { formatDate } from '@/lib/utils'

// Mock data
const mockServices: InferenceService[] = [
  { id: '1', name: 'LLaMA-2-7B Chat', status: 'running', endpoint: 'https://api.aitip.io/v1/llama2-7b', modelName: 'llama-2-7b-chat', version: 'v1.0.0', createdAt: '2024-01-10T08:00:00Z', updatedAt: '2024-01-15T10:30:00Z', requestsPerSecond: 45.2, avgLatency: 125, replicas: 3, gpuUsage: 68 },
  { id: '2', name: 'BERT Sentiment Analysis', status: 'running', endpoint: 'https://api.aitip.io/v1/bert-sentiment', modelName: 'bert-base-sentiment', version: 'v2.1.0', createdAt: '2024-01-08T06:00:00Z', updatedAt: '2024-01-15T09:45:00Z', requestsPerSecond: 120.5, avgLatency: 45, replicas: 2, gpuUsage: 35 },
  { id: '3', name: 'Stable Diffusion XL', status: 'running', endpoint: 'https://api.aitip.io/v1/sdxl', modelName: 'stable-diffusion-xl', version: 'v1.0.0', createdAt: '2024-01-12T10:00:00Z', updatedAt: '2024-01-15T10:00:00Z', requestsPerSecond: 8.3, avgLatency: 2500, replicas: 4, gpuUsage: 85 },
  { id: '4', name: 'Whisper ASR', status: 'stopped', endpoint: 'https://api.aitip.io/v1/whisper', modelName: 'whisper-large-v3', version: 'v1.2.0', createdAt: '2024-01-05T12:00:00Z', updatedAt: '2024-01-14T18:00:00Z', requestsPerSecond: 0, avgLatency: 0, replicas: 0, gpuUsage: 0 },
  { id: '5', name: 'CLIP Image Embedding', status: 'running', endpoint: 'https://api.aitip.io/v1/clip', modelName: 'clip-vit-large', version: 'v1.0.1', createdAt: '2024-01-11T09:00:00Z', updatedAt: '2024-01-15T08:30:00Z', requestsPerSecond: 85.7, avgLatency: 78, replicas: 2, gpuUsage: 42 },
  { id: '6', name: 'GPT-3.5 Turbo Clone', status: 'error', endpoint: 'https://api.aitip.io/v1/gpt35', modelName: 'gpt-3.5-turbo', version: 'v1.0.0', createdAt: '2024-01-13T14:00:00Z', updatedAt: '2024-01-15T06:20:00Z', requestsPerSecond: 0, avgLatency: 0, replicas: 0, gpuUsage: 0 },
]

const statusColors: Record<ServiceStatus, 'default' | 'secondary' | 'destructive' | 'success' | 'warning' | 'info'> = {
  deploying: 'info',
  running: 'success',
  stopped: 'secondary',
  error: 'destructive',
}

const statusIcons = {
  deploying: RefreshCw,
  running: CheckCircle,
  stopped: Square,
  error: XCircle,
}

export function InferenceList() {
  const [services, setServices] = useState<InferenceService[]>(mockServices)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [copiedId, setCopiedId] = useState<string | null>(null)

  // Filter services
  const filteredServices = services.filter((service) => {
    const matchesSearch = service.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         service.modelName.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesStatus = statusFilter === 'all' || service.status === statusFilter
    return matchesSearch && matchesStatus
  })

  const handleCreateService = (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    setTimeout(() => {
      setIsLoading(false)
      setIsCreateDialogOpen(false)
    }, 1000)
  }

  const handleCopyEndpoint = (endpoint: string, id: string) => {
    navigator.clipboard.writeText(endpoint)
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  const handleAction = (action: string, serviceId: string) => {
    console.log(`${action} service ${serviceId}`)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Inference Services</h1>
          <p className="text-muted-foreground">Deploy and manage model serving endpoints</p>
        </div>
        <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Deploy Service
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Deploy Inference Service</DialogTitle>
              <DialogDescription>
                Configure your model serving endpoint.
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleCreateService} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Service Name</label>
                <Input placeholder="e.g., LLaMA-2 Chat API" required />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">Model</label>
                  <Select>
                    <SelectTrigger>
                      <SelectValue placeholder="Select model" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="llama2">LLaMA-2</SelectItem>
                      <SelectItem value="bert">BERT</SelectItem>
                      <SelectItem value="gpt">GPT</SelectItem>
                      <SelectItem value="sdxl">Stable Diffusion XL</SelectItem>
                      <SelectItem value="custom">Custom Model</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">Version</label>
                  <Input placeholder="v1.0.0" defaultValue="v1.0.0" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">Replicas</label>
                  <Select defaultValue="1">
                    <SelectTrigger>
                      <SelectValue placeholder="Select replicas" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1">1</SelectItem>
                      <SelectItem value="2">2</SelectItem>
                      <SelectItem value="3">3</SelectItem>
                      <SelectItem value="4">4</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">GPU per Replica</label>
                  <Select defaultValue="1">
                    <SelectTrigger>
                      <SelectValue placeholder="Select GPUs" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1">1 GPU</SelectItem>
                      <SelectItem value="2">2 GPUs</SelectItem>
                      <SelectItem value="4">4 GPUs</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setIsCreateDialogOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={isLoading}>
                  {isLoading ? <><Spinner className="mr-2" size="sm" /> Deploying...</> : 'Deploy Service'}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {/* Stats Overview */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Running Services</p>
                <p className="text-2xl font-bold">{services.filter(s => s.status === 'running').length}</p>
              </div>
              <CheckCircle className="h-8 w-8 text-green-500" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Total Requests/s</p>
                <p className="text-2xl font-bold">
                  {services.reduce((acc, s) => acc + s.requestsPerSecond, 0).toFixed(1)}
                </p>
              </div>
              <TrendingUp className="h-8 w-8 text-blue-500" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Avg Latency</p>
                <p className="text-2xl font-bold">
                  {Math.round(services.filter(s => s.status === 'running').reduce((acc, s) => acc + s.avgLatency, 0) / services.filter(s => s.status === 'running').length || 0)}ms
                </p>
              </div>
              <Clock className="h-8 w-8 text-yellow-500" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">GPU Utilization</p>
                <p className="text-2xl font-bold">
                  {Math.round(services.filter(s => s.status === 'running').reduce((acc, s) => acc + s.gpuUsage, 0) / services.filter(s => s.status === 'running').length || 0)}%
                </p>
              </div>
              <Cpu className="h-8 w-8 text-purple-500" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col gap-4 sm:flex-row">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search services..."
                className="pl-10"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[160px]">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="running">Running</SelectItem>
                <SelectItem value="stopped">Stopped</SelectItem>
                <SelectItem value="deploying">Deploying</SelectItem>
                <SelectItem value="error">Error</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Services Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>All Services</CardTitle>
              <CardDescription>{filteredServices.length} services found</CardDescription>
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
                  <TableHead>Endpoint</TableHead>
                  <TableHead>Metrics</TableHead>
                  <TableHead>Replicas</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredServices.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-24 text-center">
                      No services found
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredServices.map((service) => {
                    const StatusIcon = statusIcons[service.status]
                    return (
                      <TableRow key={service.id}>
                        <TableCell>
                          <div className="font-medium">{service.name}</div>
                          <div className="text-xs text-muted-foreground">{service.modelName} · {service.version}</div>
                        </TableCell>
                        <TableCell>
                          <Badge variant={statusColors[service.status]} className="flex w-fit items-center gap-1">
                            <StatusIcon className={`h-3 w-3 ${service.status === 'deploying' ? 'animate-spin' : ''}`} />
                            {service.status}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <code className="rounded bg-muted px-2 py-1 text-xs">{service.endpoint}</code>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-6 w-6"
                              onClick={() => handleCopyEndpoint(service.endpoint, service.id)}
                            >
                              {copiedId === service.id ? (
                                <CheckCircle className="h-3 w-3 text-green-500" />
                              ) : (
                                <Copy className="h-3 w-3" />
                              )}
                            </Button>
                          </div>
                        </TableCell>
                        <TableCell>
                          {service.status === 'running' ? (
                            <div className="space-y-1">
                              <div className="flex items-center gap-2 text-xs">
                                <TrendingUp className="h-3 w-3" />
                                <span>{service.requestsPerSecond.toFixed(1)} req/s</span>
                              </div>
                              <div className="flex items-center gap-2 text-xs">
                                <Clock className="h-3 w-3" />
                                <span>{service.avgLatency}ms latency</span>
                              </div>
                            </div>
                          ) : (
                            <span className="text-xs text-muted-foreground">No metrics</span>
                          )}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1">
                            <Users className="h-3 w-3" />
                            <span className="text-xs">{service.replicas}</span>
                            {service.status === 'running' && (
                              <span className="text-xs text-muted-foreground">({service.gpuUsage}% GPU)</span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {formatDate(service.createdAt)}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
                              <a href={service.endpoint} target="_blank" rel="noopener noreferrer">
                                <ExternalLink className="h-4 w-4" />
                              </a>
                            </Button>
                            {service.status === 'stopped' ? (
                              <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => handleAction('start', service.id)}>
                                <Play className="h-4 w-4" />
                              </Button>
                            ) : service.status === 'running' ? (
                              <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => handleAction('stop', service.id)}>
                                <Square className="h-4 w-4" />
                              </Button>
                            ) : null}
                            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => handleAction('settings', service.id)}>
                              <Settings className="h-4 w-4" />
                            </Button>
                            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => handleAction('delete', service.id)}>
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
