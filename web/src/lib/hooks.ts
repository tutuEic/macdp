// React Query hooks for MACDP API
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as api from './api'
import type { CreateProjectInput, CreateTaskInput, PlanInput, ChatInput } from './types'

// ---- Projects ----

export function useProjects() {
  return useQuery({ queryKey: ['projects'], queryFn: api.listProjects })
}

export function useProject(id: string) {
  return useQuery({ queryKey: ['projects', id], queryFn: () => api.getProject(id), enabled: !!id })
}

export function useCreateProject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateProjectInput) => api.createProject(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['projects'] }),
  })
}

export function usePlanProject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: PlanInput }) => api.planProject(id, input),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ['projects', id] })
      qc.invalidateQueries({ queryKey: ['tasks', id] })
    },
  })
}

export function useExecuteProject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.executeProject(id),
    onSuccess: (_, id) => qc.invalidateQueries({ queryKey: ['tasks', id] }),
  })
}

// ---- Tasks ----

export function useTasks(projectId: string) {
  return useQuery({
    queryKey: ['tasks', projectId],
    queryFn: () => api.listTasks(projectId),
    enabled: !!projectId,
    refetchInterval: 3000, // poll every 3s for real-time feel
  })
}

export function useCreateTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ projectId, input }: { projectId: string; input: CreateTaskInput }) =>
      api.createTask(projectId, input),
    onSuccess: (_, { projectId }) => qc.invalidateQueries({ queryKey: ['tasks', projectId] }),
  })
}

export function useUpdateTaskStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) => api.updateTaskStatus(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks'] }),
  })
}

// ---- Agents ----

export function useAgents() {
  return useQuery({
    queryKey: ['agents'],
    queryFn: api.listAgents,
    refetchInterval: 5000,
  })
}

export function useAgentStatus(name: string) {
  return useQuery({
    queryKey: ['agents', name],
    queryFn: () => api.getAgentStatus(name),
    enabled: !!name,
    refetchInterval: 3000,
  })
}

export function useSendAgentMessage() {
  return useMutation({
    mutationFn: ({ name, message }: { name: string; message: string }) =>
      api.sendAgentMessage(name, message),
  })
}

// ---- Chat ----

export function useChatHistory(projectId: string, agent?: string) {
  return useQuery({
    queryKey: ['chat', projectId, agent],
    queryFn: () => api.getChatHistory(projectId, agent),
    enabled: !!projectId,
    refetchInterval: 2000,
  })
}

export function useSendChatMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ projectId, input }: { projectId: string; input: ChatInput }) =>
      api.sendChatMessage(projectId, input),
    onSuccess: (_, { projectId }) => qc.invalidateQueries({ queryKey: ['chat', projectId] }),
  })
}

// ---- Events ----

export function useEvents(type?: string) {
  return useQuery({
    queryKey: ['events', type],
    queryFn: () => api.getEvents(type),
    refetchInterval: 3000,
  })
}
