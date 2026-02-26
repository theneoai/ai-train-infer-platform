export function ExperimentList() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Experiments</h1>
          <p className="text-muted-foreground">Track and compare your ML experiments</p>
        </div>
        <button className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90">
          New Experiment
        </button>
      </div>

      <div className="rounded-lg border">
        <div className="p-4">
          <p className="text-sm text-muted-foreground">No experiments found</p>
        </div>
      </div>
    </div>
  )
}
