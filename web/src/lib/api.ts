// API client for MACDP backend
import type {
  Project, Task, AgentInfo, ChatMessage, TaskEvent,
  PlanResult, CreateProjectInput, CreateTaskInput, PlanInput, ChatInput,
} from './types'

const BASE = '/api'

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(BASE + url, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || `HTTP ${res.status}`)
  }
  return res.json()
}

// ---- Projects ----

export function listProjects() {
  return request<Project[]>('/projects')
}

export function getProject(id: string) {
  return request<Project>(`/projects/${id}`)
}

export function createProject(input: CreateProjectInput) {
  return request<Project>('/projects', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function planProject(id: string, input: PlanInput) {
  return request<PlanResult>(`/projects/${id}/plan`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function executeProject(id: string) {
  return request<{ status: string; tasks: number }>(`/projects/${id}/execute`, {
    method: 'POST',
  })
}

// ---- Tasks ----

export function listTasks(projectId: string) {
  return request<Task[]>(`/projects/${projectId}/tasks`)
}

export function getTask(id: string) {
  return request<Task>(`/tasks/${id}`)
}

export function createTask(projectId: string, input: CreateTaskInput) {
  return request<Task>(`/projects/${projectId}/tasks`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateTaskStatus(id: string, status: string) {
  return request<{ status: string }>(`/tasks/${id}/status`, {
    method: 'PUT',
    body: JSON.stringify({ status }),
  })
}

export function assignTask(id: string, agent: string) {
  return request<{ status: string }>(`/tasks/${id}/assign/${agent}`, {
    method: 'PUT',
  })
}

// ---- Agents ----

export function listAgents() {
  return request<AgentInfo[]>('/agents')
}

export function getAgentStatus(name: string) {
  return request<AgentInfo>(`/agents/${name}/status`)
}

export function sendAgentMessage(name: string, message: string) {
  return request<ChatMessage>(`/agents/${name}/message`, {
    method: 'POST',
    body: JSON.stringify({ message }),
  })
}

// ---- Chat ----

export function getChatHistory(projectId: string, agent?: string) {
  const params = agent ? `?agent=${agent}` : ''
  return request<ChatMessage[]>(`/projects/${projectId}/chat${params}`)
}

export function sendChatMessage(projectId: string, input: ChatInput) {
  return request<ChatMessage>(`/projects/${projectId}/chat`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

// ---- Events ----

export function getEvents(type?: string) {
  const params = type ? `?type=${type}` : ''
  return request<TaskEvent[]>(`/events${params}`)
}

// ---- Memory ----

export function getMemoryStats(projectId: string) {
  return request<any>(`/projects/${projectId}/memory/stats`)
}

export function getMemoryEntries(projectId: string, module?: string, category?: string) {
  const params = new URLSearchParams()
  if (module) params.set('module', module)
  if (category) params.set('category', category)
  return request<any[]>(`/projects/${projectId}/memory/entries?${params}`)
}

// ---- Health ----

export function healthCheck() {
  return request<{ status: string }>('/health')
}