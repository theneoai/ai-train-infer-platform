import { useParams } from '@tanstack/react-router'

export function TrainingDetail() {
  const { jobId } = useParams({ from: '/training/$jobId' })

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Training Job: {jobId}</h1>
        <p className="text-muted-foreground">View training job details and logs</p>
      </div>

      <div className="rounded-lg border p-4">
        <p className="text-sm text-muted-foreground">Job details will be displayed here</p>
      </div>
    </div>
  )
}
