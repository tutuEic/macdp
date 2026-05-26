import { useState } from 'react'

interface Task {
  id: string
  title: string
  agent: string
  module: string
  progress: number
  status: 'pending' | 'running' | 'review' | 'done' | 'failed'
}

export default function Kanban() {
  const [tasks] = useState<Task[]>([
    { id: 'T-001', title: 'Project Setup', agent: 'Hermes', module: 'devops', progress: 100, status: 'done' },
    { id: 'T-002', title: 'Database Schema', agent: 'Hermes', module: 'database', progress: 100, status: 'done' },
    { id: 'T-003', title: 'Login API', agent: 'Claude', module: 'backend', progress: 60, status: 'running' },
    { id: 'T-004', title: 'React Init', agent: 'Codex', module: 'frontend', progress: 100, status: 'done' },
    { id: 'T-005', title: 'JWT Middleware', agent: 'Claude', module: 'backend', progress: 0, status: 'pending' },
    { id: 'T-006', title: 'Login Page', agent: 'Codex', module: 'frontend', progress: 0, status: 'pending' },
    { id: 'T-007', title: 'Register API', agent: 'Claude', module: 'backend', progress: 0, status: 'pending' },
    { id: 'T-008', title: 'Integration Tests', agent: 'Hermes', module: 'testing', progress: 0, status: 'pending' },
  ])

  const columns = [
    { key: 'pending', label: 'To Do', color: 'var(--text-muted)' },
    { key: 'running', label: 'In Progress', color: 'var(--accent-gold)' },
    { key: 'review', label: 'Review', color: 'var(--info)' },
    { key: 'done', label: 'Done', color: 'var(--success)' },
  ]

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Kanban</h1>
          <p className="page-subtitle">Task board for Todo App</p>
        </div>
        <button className="btn btn-primary">+ New Task</button>
      </div>

      <div className="kanban-board">
        {columns.map(col => {
          const colTasks = tasks.filter(t => t.status === col.key)
          return (
            <div key={col.key} className="kanban-column">
              <div className="kanban-column-header">
                <span className="kanban-column-title" style={{color: col.color}}>{col.label}</span>
                <span className="kanban-column-count">{colTasks.length}</span>
              </div>
              {colTasks.map(task => (
                <div key={task.id} className="card task-card">
                  <div className="task-card-title">{task.title}</div>
                  <div style={{fontSize:12, color:'var(--text-muted)', marginBottom:8}}>
                    {task.id} · {task.module}
                  </div>
                  {task.status === 'running' && (
                    <div className="progress-bar" style={{marginBottom:8}}>
                      <div className="progress-bar-fill" style={{width:`${task.progress}%`}}></div>
                    </div>
                  )}
                  <div className="task-card-meta">
                    <div className="task-card-agent">
                      <span className="status-dot online" style={{width:6, height:6}}></span>
                      {task.agent}
                    </div>
                    {task.status === 'running' && <span style={{color:'var(--accent-gold)', fontWeight:500}}>{task.progress}%</span>}
                    {task.status === 'done' && <span style={{color:'var(--success)'}}>✓</span>}
                  </div>
                </div>
              ))}
            </div>
          )
        })}
      </div>
    </div>
  )
}
