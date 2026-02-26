export function TrainingList() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Training Jobs</h1>
          <p className="text-muted-foreground">Manage your AI training workloads</p>
        </div>
        <button className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90">
          New Training Job
        </button>
      </div>

      <div className="rounded-lg border">
        <div className="p-4">
          <p className="text-sm text-muted-foreground">No training jobs found</p>
        </div>
      </div>
    </div>
  )
}
