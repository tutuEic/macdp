import { useState } from 'react'

interface Project {
  id: string
  name: string
  description: string
  tasks: number
  completed: number
  agents: string[]
}

export default function Projects() {
  const [projects] = useState<Project[]>([
    { id: '1', name: 'Todo App', description: 'FastAPI + React todo application with JWT authentication', tasks: 8, completed: 5, agents: ['Hermes', 'Claude', 'Codex'] },
    { id: '2', name: 'Analytics Dashboard', description: 'Real-time data analytics dashboard with WebSocket', tasks: 12, completed: 3, agents: ['Claude', 'Codex'] },
  ])

  const [showNew, setShowNew] = useState(false)

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Projects</h1>
          <p className="page-subtitle">Manage your development projects</p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowNew(!showNew)}>
          + New Project
        </button>
      </div>

      {showNew && (
        <div className="card animate-fade-in" style={{padding:24, marginBottom:24}}>
          <h3 style={{fontSize:16, marginBottom:16, fontFamily:'Inter', fontWeight:600}}>Create Project</h3>
          <div style={{display:'flex', flexDirection:'column', gap:12}}>
            <input className="input" placeholder="Project name" />
            <textarea className="input" placeholder="Description" rows={3} style={{resize:'vertical'}} />
            <input className="input" placeholder="Repository path (e.g., /mnt/d/projects/myapp)" />
            <div style={{display:'flex', gap:8, justifyContent:'flex-end'}}>
              <button className="btn btn-secondary" onClick={() => setShowNew(false)}>Cancel</button>
              <button className="btn btn-primary">Create</button>
            </div>
          </div>
        </div>
      )}

      <div className="project-grid">
        {projects.map(p => (
          <div key={p.id} className="card project-card">
            <div>
              <div className="project-name">{p.name}</div>
              <div className="project-desc">{p.description}</div>
            </div>
            <div className="project-stats">
              <div className="project-stat">
                <span className="project-stat-value">{p.tasks}</span>
                <span className="project-stat-label">Tasks</span>
              </div>
              <div className="project-stat">
                <span className="project-stat-value">{p.completed}</span>
                <span className="project-stat-label">Done</span>
              </div>
              <div className="project-stat">
                <span className="project-stat-value">{Math.round(p.completed/p.tasks*100)}%</span>
                <span className="project-stat-label">Progress</span>
              </div>
            </div>
            <div style={{display:'flex', gap:6, flexWrap:'wrap'}}>
              {p.agents.map(a => (
                <span key={a} className="badge badge-info">{a}</span>
              ))}
            </div>
            <div className="progress-bar">
              <div className="progress-bar-fill" style={{width:`${p.completed/p.tasks*100}%`}}></div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
