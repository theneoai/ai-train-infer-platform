export function Dashboard() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          AI Train-Infer-Sim Platform Overview
        </p>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {[
          { title: 'Active Training Jobs', value: '12', change: '+3' },
          { title: 'Inference Services', value: '8', change: '+1' },
          { title: 'Simulations Running', value: '5', change: '0' },
          { title: 'GPU Utilization', value: '78%', change: '+5%' },
        ].map((stat) => (
          <div key={stat.title} className="rounded-lg border bg-card p-4">
            <p className="text-sm font-medium text-muted-foreground">{stat.title}</p>
            <div className="mt-2 flex items-baseline gap-2">
              <p className="text-2xl font-bold">{stat.value}</p>
              <span className="text-xs text-green-600">{stat.change}</span>
            </div>
          </div>
        ))}
      </div>

      {/* Recent Activity */}
      <div className="rounded-lg border">
        <div className="border-b px-4 py-3">
          <h3 className="font-semibold">Recent Activity</h3>
        </div>
        <div className="p-4">
          <p className="text-sm text-muted-foreground">No recent activity</p>
        </div>
      </div>
    </div>
  )
}
