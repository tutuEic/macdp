import { useAgents, useEvents, useProjects, useWebSocket } from '../lib'
import { useState } from 'react'
import type { TaskEvent } from '../lib/types'

export default function Dashboard() {
  const { data: agents, isLoading: agentsLoading } = useAgents()
  const { data: events, isLoading: eventsLoading } = useEvents()
  const { data: projects, isLoading: projectsLoading } = useProjects()
  const [wsEvents, setWsEvents] = useState<TaskEvent[]>([])

  // Real-time WebSocket events
  useWebSocket({
    onEvent: (ev) => {
      setWsEvents(prev => [ev, ...prev].slice(0, 20))
    },
  })

  const onlineCount = agents?.filter((a: any) => a.online).length ?? 0
  const totalAgents = agents?.length ?? 0
  const activeTasks = agents?.filter((a: any) => a.state === 'busy').length ?? 0
  const totalProjects = projects?.length ?? 0

  // Build activity feed from events + WebSocket
  const allEvents = [...wsEvents.slice(0, 5), ...(events ?? [])].slice(0, 10)

  const getActivityIcon = (type: string) => {
    if (type.includes('completed')) return '✓'
    if (type.includes('started')) return '▶'
    if (type.includes('failed')) return '✗'
    if (type.includes('created')) return '+'
    if (type.includes('message')) return '◎'
    return '◉'
  }

  const formatPayload = (ev: TaskEvent) => {
    const p = ev.payload
    if (typeof p === 'string') return p
    if (p?.title) return `${p.title}${p.agent ? ` by ${p.agent}` : ''}`
    return ev.type
  }

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
          <span className="stat-value">{agentsLoading ? '...' : onlineCount}
            <span style={{ fontSize: 16, color: 'var(--text-muted)' }}>/{totalAgents}</span>
          </span>
          <span className="stat-change">{activeTasks > 0 ? `${activeTasks} active tasks` : 'All idle'}</span>
        </div>
        <div className="card stat-card">
          <span className="stat-label">Projects</span>
          <span className="stat-value">{projectsLoading ? '...' : totalProjects}</span>
          <span className="stat-change">Total projects</span>
        </div>
        <div className="card stat-card">
          <span className="stat-label">Recent Events</span>
          <span className="stat-value">{allEvents.length}</span>
          <span className="stat-change">Last 24h</span>
        </div>
        <div className="card stat-card">
          <span className="stat-label">WebSocket</span>
          <span className="stat-value" style={{ color: wsEvents.length > 0 ? 'var(--success)' : 'var(--text-muted)' }}>
            {wsEvents.length > 0 ? 'Live' : 'Idle'}
          </span>
          <span className="stat-change">{wsEvents.length} events received</span>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px' }}>
        {/* Agent Status */}
        <div className="card" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: 16, marginBottom: 16, fontFamily: 'Inter', fontWeight: 600 }}>
            Agent Status {agentsLoading && '(loading...)'}
          </h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            {agents?.map(agent => (
              <div key={agent.name} style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                padding: '12px 16px', background: 'var(--bg-secondary)', borderRadius: 'var(--radius-sm)',
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <span className={`status-dot ${agent.online ? (agent.state === 'busy' ? 'busy' : 'online') : 'offline'}`} />
                  <div>
                    <div style={{ fontSize: 14, fontWeight: 500 }}>{agent.name}</div>
                    <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{agent.type}</div>
                  </div>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <span className={`badge ${agent.state === 'idle' ? 'badge-success' : agent.state === 'busy' ? 'badge-warning' : 'badge-idle'}`}>
                    {agent.state}
                  </span>
                  {agent.task && <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>{agent.task}</div>}
                </div>
              </div>
            ))}
            {agentsLoading && (agents ?? []).length === 0 && (
              <div style={{ color: 'var(--text-muted)', padding: '20px', textAlign: 'center' }}>Connecting to backend...</div>
            )}
          </div>
        </div>

        {/* Activity Feed */}
        <div className="card" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: 16, marginBottom: 16, fontFamily: 'Inter', fontWeight: 600 }}>Activity</h3>
          <div className="activity-feed">
            {allEvents.map((ev, i) => (
              <div key={i} className="activity-item">
                <div className={`activity-icon ${ev.type.includes('completed') ? 'success' : ev.type.includes('failed') ? 'error' : ''}`}>
                  {getActivityIcon(ev.type)}
                </div>
                <div className="activity-content">
                  <div className="activity-text">{formatPayload(ev)}</div>
                  <div className="activity-time">{new Date(ev.timestamp).toLocaleTimeString()}</div>
                </div>
              </div>
            ))}
            {allEvents.length === 0 && !eventsLoading && (
              <div style={{ color: 'var(--text-muted)', padding: '20px', textAlign: 'center' }}>
                No events yet. Create a project to get started.
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
