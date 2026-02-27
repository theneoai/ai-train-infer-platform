import { createRootRoute, Link, Outlet } from '@tanstack/react-router'
import { useAuthStore } from '../stores/auth'
import { ProtectedRoute } from '../components/ProtectedRoute'
import { 
  LayoutDashboard, 
  Cpu, 
  Database, 
  Server,
  LogOut,
  User
} from 'lucide-react'

export const rootRoute = createRootRoute({
  component: () => {
    const { user, logout } = useAuthStore()
    const isLoginPage = window.location.pathname === '/login'

    if (isLoginPage) {
      return <Outlet />
    }

    return (
      <ProtectedRoute>
        <div className="min-h-screen bg-gray-50">
          {/* Header */}
          <header className="bg-white border-b">
            <div className="flex h-16 items-center px-6">
              <div className="flex items-center gap-2">
                <div className="h-8 w-8 bg-blue-600 rounded-lg flex items-center justify-center">
                  <span className="text-white font-bold">AI</span>
                </div>
                <span className="text-xl font-semibold">AITIP</span>
              </div>

              <nav className="flex items-center gap-6 ml-10">
                <Link to="/" className="flex items-center gap-2 text-gray-600 hover:text-gray-900">
                  <LayoutDashboard className="h-4 w-4" />
                  Dashboard
                </Link>
                <Link to="/training" className="flex items-center gap-2 text-gray-600 hover:text-gray-900">
                  <Cpu className="h-4 w-4" />
                  Training
                </Link>
                <Link to="/datasets" className="flex items-center gap-2 text-gray-600 hover:text-gray-900">
                  <Database className="h-4 w-4" />
                  Datasets
                </Link>
                <Link to="/inference" className="flex items-center gap-2 text-gray-600 hover:text-gray-900">
                  <Server className="h-4 w-4" />
                  Inference
                </Link>
              </nav>

              <div className="ml-auto flex items-center gap-4">
                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <User className="h-4 w-4" />
                  {user?.email}
                </div>
                <button 
                  onClick={logout}
                  className="flex items-center gap-2 text-sm text-gray-600 hover:text-red-600"
                >
                  <LogOut className="h-4 w-4" />
                  Logout
                </button>
              </div>
            </div>
          </header>

          {/* Main Content */}
          <main className="p-6">
            <Outlet />
          </main>
        </div>
      </ProtectedRoute>
    )
  },
})
