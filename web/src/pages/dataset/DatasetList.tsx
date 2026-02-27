import { useState, useRef, useEffect } from 'react'
import {
  Upload,
  FileText,
  Trash2,
  HardDrive,
  FileType,
  Calendar,
  Search,
  AlertCircle,
  RefreshCw,
  X,
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
import { Spinner } from '@/components/ui/spinner'
import { formatDate, formatBytes } from '@/lib/utils'
import { datasetApi } from '@/services/api'

interface Dataset {
  id: string
  name: string
  size: number
  format: string
  fileCount: number
  createdAt: string
  updatedAt: string
  description?: string
}

const formatLabels: Record<string, string> = {
  'csv': 'CSV',
  'json': 'JSON',
  'jsonl': 'JSONL',
  'parquet': 'Parquet',
  'txt': 'Text',
  'zip': 'ZIP',
  'tar': 'TAR',
}

const formatColors: Record<string, 'default' | 'secondary' | 'destructive' | 'success' | 'warning' | 'info'> = {
  'csv': 'success',
  'json': 'info',
  'jsonl': 'info',
  'parquet': 'warning',
  'txt': 'secondary',
  'zip': 'default',
  'tar': 'default',
}

export function DatasetList() {
  const [datasets, setDatasets] = useState<Dataset[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [isUploadDialogOpen, setIsUploadDialogOpen] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [datasetName, setDatasetName] = useState('')
  const [datasetDescription, setDatasetDescription] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Fetch datasets
  const fetchDatasets = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await datasetApi.list() as unknown as { datasets: Dataset[] } | Dataset[]
      if (Array.isArray(response)) {
        setDatasets(response)
      } else if (response && 'datasets' in response) {
        setDatasets(response.datasets)
      } else {
        setDatasets([])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取数据集失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchDatasets()
  }, [])

  // Filter datasets
  const filteredDatasets = datasets.filter((dataset) =>
    dataset.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    dataset.format.toLowerCase().includes(searchQuery.toLowerCase())
  )

  // Handle file selection
  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setSelectedFile(file)
      // Auto-fill name from filename (without extension)
      const nameWithoutExt = file.name.replace(/\.[^/.]+$/, '')
      setDatasetName(nameWithoutExt)
    }
  }

  // Handle upload
  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedFile || !datasetName) return

    setIsUploading(true)
    setUploadProgress(0)

    try {
      const formData = new FormData()
      formData.append('file', selectedFile)
      formData.append('name', datasetName)
      formData.append('description', datasetDescription)

      await datasetApi.create(formData)

      // Reset and refresh
      setSelectedFile(null)
      setDatasetName('')
      setDatasetDescription('')
      setIsUploadDialogOpen(false)
      fetchDatasets()
    } catch (err) {
      alert(err instanceof Error ? err.message : '上传失败')
    } finally {
      setIsUploading(false)
      setUploadProgress(0)
    }
  }

  // Handle delete
  const handleDelete = async (id: string) => {
    if (!confirm('确定要删除这个数据集吗？此操作不可恢复。')) return

    try {
      await datasetApi.delete(id)
      fetchDatasets()
    } catch (err) {
      alert(err instanceof Error ? err.message : '删除失败')
    }
  }

  // Get file icon based on format
  const getFileIcon = (_format: string) => {
    const colorClass = 'text-muted-foreground'
    return <FileText className={`h-5 w-5 ${colorClass}`} />
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">数据集管理</h1>
          <p className="text-muted-foreground">管理和上传训练数据集</p>
        </div>
        <Button onClick={() => setIsUploadDialogOpen(true)}>
          <Upload className="mr-2 h-4 w-4" />
          上传数据集
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col gap-4 sm:flex-row">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="搜索数据集名称或格式..."
                className="pl-10"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
            <Button variant="outline" size="icon" onClick={fetchDatasets} disabled={loading}>
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总数据集数</CardTitle>
            <HardDrive className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{datasets.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总存储</CardTitle>
            <FileType className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatBytes(datasets.reduce((acc, ds) => acc + ds.size, 0))}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">本月上传</CardTitle>
            <Calendar className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {datasets.filter(ds => {
                const created = new Date(ds.createdAt)
                const now = new Date()
                return created.getMonth() === now.getMonth() && created.getFullYear() === now.getFullYear()
              }).length}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Dataset Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>数据集列表</CardTitle>
              <CardDescription>共 {filteredDatasets.length} 个数据集</CardDescription>
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
                  <TableHead>名称</TableHead>
                  <TableHead>格式</TableHead>
                  <TableHead>文件数</TableHead>
                  <TableHead>大小</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={6} className="h-24 text-center">
                      <div className="flex items-center justify-center gap-2">
                        <Spinner size="sm" />
                        加载中...
                      </div>
                    </TableCell>
                  </TableRow>
                ) : filteredDatasets.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="h-24 text-center">
                      <div className="flex flex-col items-center justify-center gap-2 text-muted-foreground">
                        <HardDrive className="h-8 w-8" />
                        <p>暂无数据集</p>
                        <Button variant="outline" size="sm" onClick={() => setIsUploadDialogOpen(true)}>
                          <Upload className="mr-2 h-4 w-4" />
                          上传数据集
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredDatasets.map((dataset) => (
                    <TableRow key={dataset.id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          {getFileIcon(dataset.format)}
                          <div>
                            <div className="font-medium">{dataset.name}</div>
                            {dataset.description && (
                              <div className="text-xs text-muted-foreground truncate max-w-[200px]">
                                {dataset.description}
                              </div>
                            )}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={formatColors[dataset.format] || 'default'}>
                          {formatLabels[dataset.format] || dataset.format.toUpperCase()}
                        </Badge>
                      </TableCell>
                      <TableCell>{dataset.fileCount || 1} 个文件</TableCell>
                      <TableCell>{formatBytes(dataset.size)}</TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatDate(dataset.createdAt)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          onClick={() => handleDelete(dataset.id)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Upload Dialog */}
      <Dialog open={isUploadDialogOpen} onOpenChange={setIsUploadDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>上传数据集</DialogTitle>
            <DialogDescription>
              上传您的训练数据集文件。支持 CSV, JSON, JSONL, Parquet 等格式。
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleUpload} className="space-y-4">
            {/* File Upload Area */}
            <div
              className={`rounded-lg border-2 border-dashed p-8 text-center transition-colors ${
                selectedFile ? 'border-primary bg-primary/5' : 'border-muted-foreground/25 hover:border-primary/50'
              }`}
              onClick={() => fileInputRef.current?.click()}
            >
              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                onChange={handleFileSelect}
                accept=".csv,.json,.jsonl,.parquet,.txt,.zip,.tar"
              />
              {selectedFile ? (
                <div className="space-y-2">
                  <FileText className="mx-auto h-8 w-8 text-primary" />
                  <p className="font-medium">{selectedFile.name}</p>
                  <p className="text-sm text-muted-foreground">{formatBytes(selectedFile.size)}</p>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation()
                      setSelectedFile(null)
                      setDatasetName('')
                    }}
                  >
                    <X className="mr-1 h-4 w-4" />
                    移除
                  </Button>
                </div>
              ) : (
                <div className="space-y-2">
                  <Upload className="mx-auto h-8 w-8 text-muted-foreground" />
                  <p className="font-medium">点击或拖拽文件到此处</p>
                  <p className="text-sm text-muted-foreground">支持 CSV, JSON, JSONL, Parquet, ZIP, TAR</p>
                </div>
              )}
            </div>

            {/* Dataset Info */}
            <div className="space-y-2">
              <label className="text-sm font-medium">数据集名称 *</label>
              <Input
                placeholder="输入数据集名称"
                required
                value={datasetName}
                onChange={(e) => setDatasetName(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">描述（可选）</label>
              <Input
                placeholder="输入数据集描述"
                value={datasetDescription}
                onChange={(e) => setDatasetDescription(e.target.value)}
              />
            </div>

            {/* Upload Progress */}
            {isUploading && (
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span>上传中...{uploadProgress}%</span>
                </div>
                <div className="h-2 rounded-full bg-muted">
                  <div
                    className="h-2 rounded-full bg-primary transition-all"
                    style={{ width: `${uploadProgress}%` }}
                  />
                </div>
              </div>
            )}

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsUploadDialogOpen(false)}
                disabled={isUploading}
              >
                取消
              </Button>
              <Button
                type="submit"
                disabled={!selectedFile || !datasetName || isUploading}
              >
                {isUploading ? (
                  <><Spinner className="mr-2" size="sm" />上传中...</>
                ) : '开始上传'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
