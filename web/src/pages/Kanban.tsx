import { useState } from 'react'
import { useTasks, useUpdateTaskStatus, useProjects, useCreateTask } from '../lib'
import type { TaskStatus } from '../lib/types'

export default function Kanban() {
  const { data: projects } = useProjects()
  const [selectedProject, setSelectedProject] = useState('')
  const { data: tasks, isLoading } = useTasks(selectedProject)
  const updateStatus = useUpdateTaskStatus()
  const createTask = useCreateTask()
  const [showNew, setShowNew] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [newModule, setNewModule] = useState('backend')

  const columns: { key: TaskStatus; label: string; color: string }[] = [
    { key: 'pending', label: 'To Do', color: 'var(--text-muted)' },
    { key: 'assigned', label: 'Assigned', color: 'var(--info)' },
    { key: 'running', label: 'In Progress', color: 'var(--accent-gold)' },
    { key: 'review', label: 'Review', color: '#9b59b6' },
    { key: 'done', label: 'Done', color: 'var(--success)' },
  ]

  const handleStatusChange = async (taskId: string, newStatus: TaskStatus) => {
    try {
      await updateStatus.mutateAsync({ id: taskId, status: newStatus })
    } catch (e: any) {
      alert(`Failed: ${e.message}`)
    }
  }

  const handleCreateTask = async () => {
    if (!newTitle.trim() || !selectedProject) return
    try {
      await createTask.mutateAsync({
        projectId: selectedProject,
        input: { title: newTitle, description: newDesc, module: newModule },
      })
      setNewTitle(''); setNewDesc(''); setShowNew(false)
    } catch (e: any) {
      alert(`Failed: ${e.message}`)
    }
  }

  const taskByStatus = (status: TaskStatus) => tasks?.filter(t => t.status === status) ?? []

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Kanban</h1>
          <p className="page-subtitle">Task board</p>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <select
            value={selectedProject}
            onChange={e => setSelectedProject(e.target.value)}
            style={{
              padding: '8px 12px', borderRadius: 8, border: '1px solid var(--border-light)',
              fontFamily: 'Inter', fontSize: 13, background: 'var(--bg-card)',
            }}
          >
            <option value="">Select project...</option>
            {projects?.map(p => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>
          <button className="btn btn-primary" onClick={() => setShowNew(true)} disabled={!selectedProject}>
            + New Task
          </button>
        </div>
      </div>

      {/* New task modal */}
      {showNew && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)',
          display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000,
        }} onClick={() => setShowNew(false)}>
          <div className="card" style={{ width: 400, padding: 24 }} onClick={e => e.stopPropagation()}>
            <h3 style={{ fontSize: 16, marginBottom: 12, fontWeight: 600 }}>New Task</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              <input className="input" placeholder="Title" value={newTitle} onChange={e => setNewTitle(e.target.value)} />
              <textarea className="input" placeholder="Description" rows={2} value={newDesc} onChange={e => setNewDesc(e.target.value)} />
              <select className="input" value={newModule} onChange={e => setNewModule(e.target.value)}>
                <option value="backend">Backend</option>
                <option value="frontend">Frontend</option>
                <option value="database">Database</option>
                <option value="testing">Testing</option>
                <option value="devops">DevOps</option>
              </select>
              <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
                <button className="btn btn-secondary" onClick={() => setShowNew(false)}>Cancel</button>
                <button className="btn btn-primary" onClick={handleCreateTask} disabled={createTask.isPending}>
                  Create
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {isLoading && (
        <div style={{ display: 'flex', justifyContent: 'center', padding: 40 }}>
          <div className="loading-spinner" />
        </div>
      )}

      {!selectedProject && (
        <div className="card" style={{ padding: 40, textAlign: 'center', color: 'var(--text-muted)' }}>
          Select a project to view its task board
        </div>
      )}

      {selectedProject && (
        <div className="kanban-board">
          {columns.map(col => {
            const colTasks = taskByStatus(col.key)
            return (
              <div key={col.key} className="kanban-column">
                <div className="kanban-column-header">
                  <span className="kanban-column-title" style={{ color: col.color }}>{col.label}</span>
                  <span className="kanban-column-count">{colTasks.length}</span>
                </div>
                {colTasks.map(task => (
                  <div key={task.id} className="card task-card">
                    <div className="task-card-title">{task.title}</div>
                    <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 8 }}>
                      {task.id} · {task.module}
                    </div>

                    {task.status === 'running' && (
                      <div className="progress-bar" style={{ marginBottom: 8 }}>
                        <div className="progress-bar-fill" style={{ width: `${task.progress}%` }} />
                      </div>
                    )}

                    <div className="task-card-meta">
                      <div className="task-card-agent">
                        {task.assigned_agent ? (
                          <>
                            <span className="status-dot online" style={{ width: 6, height: 6 }} />
                            {task.assigned_agent}
                          </>
                        ) : (
                          <span style={{ color: 'var(--text-muted)' }}>Unassigned</span>
                        )}
                      </div>
                    </div>

                    {/* Quick status change buttons */}
                    <div style={{ display: 'flex', gap: 4, marginTop: 8, flexWrap: 'wrap' }}>
                      {columns.filter(c => c.key !== task.status).slice(0, 3).map(c => (
                        <button
                          key={c.key}
                          className="btn btn-ghost"
                          style={{ fontSize: 10, padding: '3px 8px' }}
                          onClick={() => handleStatusChange(task.id, c.key)}
                          disabled={updateStatus.isPending}
                        >
                          → {c.label}
                        </button>
                      ))}
                    </div>

                    {task.files_changed && task.files_changed.length > 0 && (
                      <div style={{ marginTop: 6, fontSize: 10, color: 'var(--text-muted)' }}>
                        📄 {task.files_changed.join(', ')}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
