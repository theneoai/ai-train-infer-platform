import { createRoute } from '@tanstack/react-router'
import { rootRoute } from './__root'

// Dashboard
export const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: () => import('../pages/Dashboard').then(m => m.Dashboard),
})

// Training routes
export const trainingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/training',
  component: () => import('../pages/training/TrainingList').then(m => m.TrainingList),
})

export const trainingDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/training/$jobId',
  component: () => import('../pages/training/TrainingDetail').then(m => m.TrainingDetail),
})

// Inference routes
export const inferenceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/inference',
  component: () => import('../pages/inference/InferenceList').then(m => m.InferenceList),
})

// Simulation routes
export const simulationRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/simulation',
  component: () => import('../pages/simulation/SimulationList').then(m => m.SimulationList),
})

// Experiment routes
export const experimentsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/experiments',
  component: () => import('../pages/experiments/ExperimentList').then(m => m.ExperimentList),
})

// Agent routes
export const agentRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/agent',
  component: () => import('../pages/agent/AgentConsole').then(m => m.AgentConsole),
})
