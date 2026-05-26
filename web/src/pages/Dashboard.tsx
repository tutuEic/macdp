import { useState } from 'react'

interface AgentInfo {
  name: string
  type: string
  online: boolean
  state: string
  task: string
}

interface Activity {
  type: string
  text: string
  time: string
  icon: string
}

export default function Dashboard() {
  const [agents] = useState<AgentInfo[]>([
    { name: 'Hermes', type: 'hermes', online: true, state: 'idle', task: '' },
    { name: 'Claude Code', type: 'claude-code', online: true, state: 'busy', task: 'T-003' },
    { name: 'Codex', type: 'codex', online: true, state: 'idle', task: '' },
    { name: 'OpenClaw', type: 'openclaw', online: false, state: 'offline', task: '' },
  ])

  const activities: Activity[] = [
    { type: 'agent', text: 'Claude Code started task T-003: Implement login API', time: '2 min ago', icon: '◉' },
    { type: 'task', text: 'Task T-002 completed by Hermes: Database schema', time: '5 min ago', icon: '✓' },
    { type: 'system', text: 'Plan created: 6 tasks generated from requirement', time: '8 min ago', icon: '◇' },
    { type: 'agent', text: 'Codex completed T-004: React project init', time: '12 min ago', icon: '◉' },
    { type: 'task', text: 'Review passed for T-001: Project setup', time: '15 min ago', icon: '✓' },
  ]

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Dashboard</h1>
          <p className="page-subtitle">Overview of your multi-agent development</p>
        </div>
      </div>

      {/* Stats */}
      <div className="stats-grid">
        <div className="card stat-card">
          <span className="stat-label">Agents Online</span>
          <span className="stat-value">3<span style={{fontSize:16, color:'var(--text-muted)'}}>/4</span></span>
          <span className="stat-change">All operational</span>
        </div>
        <div className="card stat-card">
          <span className="stat-label">Active Tasks</span>
          <span className="stat-value">2</span>
          <span className="stat-change">+1 from yesterday</span>
        </div>
        <div className="card stat-card">
          <span className="stat-label">Completed Today</span>
          <span className="stat-value">8</span>
          <span className="stat-change">↑ 33% from avg</span>
        </div>
        <div className="card stat-card">
          <span className="stat-label">Total Cost</span>
          <span className="stat-value">$3.20</span>
          <span className="stat-change">Within budget</span>
        </div>
      </div>

      <div style={{display:'grid', gridTemplateColumns:'1fr 1fr', gap:'24px'}}>
        {/* Agent Status */}
        <div className="card" style={{padding:'24px'}}>
          <h3 style={{fontSize:16, marginBottom:16, fontFamily:'Inter', fontWeight:600}}>Agent Status</h3>
          <div style={{display:'flex', flexDirection:'column', gap:'12px'}}>
            {agents.map(agent => (
              <div key={agent.name} style={{display:'flex', alignItems:'center', justifyContent:'space-between', padding:'12px 16px', background:'var(--bg-secondary)', borderRadius:'var(--radius-sm)'}}>
                <div style={{display:'flex', alignItems:'center', gap:'12px'}}>
                  <span className={`status-dot ${agent.online ? (agent.state === 'busy' ? 'busy' : 'online') : 'offline'}`}></span>
                  <div>
                    <div style={{fontSize:14, fontWeight:500}}>{agent.name}</div>
                    <div style={{fontSize:12, color:'var(--text-muted)'}}>{agent.type}</div>
                  </div>
                </div>
                <div style={{textAlign:'right'}}>
                  <span className={`badge ${agent.state === 'idle' ? 'badge-success' : agent.state === 'busy' ? 'badge-warning' : 'badge-idle'}`}>
                    {agent.state}
                  </span>
                  {agent.task && <div style={{fontSize:11, color:'var(--text-muted)', marginTop:4}}>Task {agent.task}</div>}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Activity Feed */}
        <div className="card" style={{padding:'24px'}}>
          <h3 style={{fontSize:16, marginBottom:16, fontFamily:'Inter', fontWeight:600}}>Recent Activity</h3>
          <div className="activity-feed">
            {activities.map((act, i) => (
              <div key={i} className="activity-item">
                <div className={`activity-icon ${act.type}`}>{act.icon}</div>
                <div className="activity-content">
                  <div className="activity-text">{act.text}</div>
                  <div className="activity-time">{act.time}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
