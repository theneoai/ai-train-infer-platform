import { createRootRoute, Link, Outlet } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/router-devtools'
import { 
  LayoutDashboard, 
  Cpu, 
  Server, 
  FlaskConical, 
  Beaker, 
  Bot,
  User
} from 'lucide-react'

export const rootRoute = createRootRoute({
  component: () => (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b bg-card">
        <div className="flex h-14 items-center px-4">
          <div className="flex items-center gap-2 font-semibold text-lg">
            <Bot className="h-6 w-6 text-primary" />
            <span>AITIP</span>
          </div>
          <nav className="flex items-center gap-6 ml-8">
            <Link to="/" className="[&.active]:text-primary flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground">
              <LayoutDashboard className="h-4 w-4" />
              Dashboard
            </Link>
            <Link to="/training" className="[&.active]:text-primary flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground">
              <Cpu className="h-4 w-4" />
              Training
            </Link>
            <Link to="/inference" className="[&.active]:text-primary flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground">
              <Server className="h-4 w-4" />
              Inference
            </Link>
            <Link to="/simulation" className="[&.active]:text-primary flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground">
              <FlaskConical className="h-4 w-4" />
              Simulation
            </Link>
            <Link to="/experiments" className="[&.active]:text-primary flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground">
              <Beaker className="h-4 w-4" />
              Experiments
            </Link>
            <Link to="/agent" className="[&.active]:text-primary flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground">
              <Bot className="h-4 w-4" />
              Agent
            </Link>
          </nav>
          <div className="ml-auto flex items-center gap-4">
            <button className="flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground">
              <User className="h-4 w-4" />
              <span>Admin</span>
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="p-6">
        <Outlet />
      </main>

      {/* DevTools */}
      {process.env.NODE_ENV === 'development' && <TanStackRouterDevtools />}
    </div>
  ),
})
