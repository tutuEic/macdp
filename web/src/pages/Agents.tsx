import { useState } from 'react'

interface Agent {
  name: string
  type: string
  online: boolean
  state: string
  currentTask: string
  capabilities: string[]
  lastPing: string
}

export default function Agents() {
  const [agents] = useState<Agent[]>([
    { name: 'Hermes', type: 'hermes', online: true, state: 'idle', currentTask: '', capabilities: ['shell', 'file_io', 'testing', 'debugging'], lastPing: '2s ago' },
    { name: 'Claude Code', type: 'claude-code', online: true, state: 'busy', currentTask: 'T-003: Login API', capabilities: ['code_gen', 'review', 'refactoring'], lastPing: '1s ago' },
    { name: 'Codex', type: 'codex', online: true, state: 'idle', currentTask: '', capabilities: ['code_gen', 'prototyping'], lastPing: '5s ago' },
    { name: 'OpenClaw', type: 'openclaw', online: false, state: 'offline', currentTask: '', capabilities: ['multi_channel', 'skills'], lastPing: '10m ago' },
  ])

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Agents</h1>
          <p className="page-subtitle">Connected AI coding agents</p>
        </div>
        <button className="btn btn-secondary">+ Connect Agent</button>
      </div>

      <div className="agent-grid">
        {agents.map(agent => (
          <div key={agent.name} className="card agent-card">
            <div className="agent-header">
              <div style={{display:'flex', alignItems:'center', gap:10}}>
                <span className={`status-dot ${agent.online ? (agent.state === 'busy' ? 'busy' : 'online') : 'offline'}`}></span>
                <span className="agent-name">{agent.name}</span>
              </div>
              <span className={`badge ${agent.state === 'idle' ? 'badge-success' : agent.state === 'busy' ? 'badge-warning' : 'badge-idle'}`}>
                {agent.state}
              </span>
            </div>

            <div className="agent-type">{agent.type}</div>

            {agent.currentTask && (
              <div style={{fontSize:13, color:'var(--accent-gold)', fontWeight:500}}>
                ▶ {agent.currentTask}
              </div>
            )}

            <div style={{display:'flex', gap:4, flexWrap:'wrap'}}>
              {agent.capabilities.map(c => (
                <span key={c} className="badge badge-idle">{c}</span>
              ))}
            </div>

            <div style={{fontSize:12, color:'var(--text-muted)'}}>
              Last ping: {agent.lastPing}
            </div>

            <div style={{display:'flex', gap:8, marginTop:4}}>
              <button className="btn btn-ghost" style={{fontSize:12, padding:'6px 12px'}}>Chat</button>
              <button className="btn btn-ghost" style={{fontSize:12, padding:'6px 12px'}}>Configure</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
