import { createRoute, createRouter } from '@tanstack/react-router'
import { rootRoute } from './__root'
import { LoginPage } from '../pages/LoginPage'
import { Dashboard } from '../pages/Dashboard'
import { TrainingList } from '../pages/training/TrainingList'
import { TrainingDetail } from '../pages/training/TrainingDetail'
import { DatasetList } from '../pages/dataset/DatasetList'
import { InferenceList } from '../pages/inference/InferenceList'

// Login route (public)
export const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginPage,
})

// Dashboard route
export const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: Dashboard,
})

// Training routes
export const trainingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/training',
  component: TrainingList,
})

export const trainingDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/training/$jobId',
  component: TrainingDetail,
})

// Dataset routes
export const datasetRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/datasets',
  component: DatasetList,
})

// Inference routes
export const inferenceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/inference',
  component: InferenceList,
})

// Create router
export const routeTree = rootRoute.addChildren([
  loginRoute,
  dashboardRoute,
  trainingRoute,
  trainingDetailRoute,
  datasetRoute,
  inferenceRoute,
])

export const router = createRouter({
  routeTree,
  defaultPreload: 'intent',
})

// Register router for type safety
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
