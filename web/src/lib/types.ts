// API types matching Go backend models

export interface Project {
  id: string
  name: string
  description: string
  repo_path: string
  created_at: string
  updated_at: string
}

export type TaskStatus = 'pending' | 'assigned' | 'running' | 'review' | 'done' | 'failed' | 'blocked'

export interface Task {
  id: string
  project_id: string
  title: string
  description: string
  module: string
  status: TaskStatus
  priority: number
  assigned_agent: string
  reviewer: string
  depends_on: string[]
  branch: string
  worktree: string
  progress: number
  output: string
  files_changed: string[]
  cost_usd: number
  max_turns: number
  started_at?: string
  completed_at?: string
  created_at: string
}

export interface AgentInfo {
  name: string
  type: string
  online: boolean
  state: string
  task: string
}

export interface ChatMessage {
  id: string
  project_id: string
  task_id?: string
  agent_name: string
  role: 'user' | 'agent' | 'system'
  content: string
  created_at: string
}

export interface TaskEvent {
  type: string
  source: string
  target: string
  payload: any
  timestamp: string
}

export interface PlanResult {
  tasks: PlannedTask[]
  summary: string
}

export interface PlannedTask {
  id: string
  title: string
  description: string
  module: string
  agent: string
  depends_on: string[]
  priority: number
  max_turns: number
}

export interface CreateProjectInput {
  name: string
  description: string
  repo_path: string
}

export interface CreateTaskInput {
  title: string
  description: string
  module: string
  depends_on?: string[]
  priority?: number
  max_turns?: number
}

export interface PlanInput {
  requirement: string
}

export interface ChatInput {
  agent_name: string
  content: string
  task_id?: string
}
