import { useAgents, useSendAgentMessage } from '../lib'
import { useState } from 'react'

export default function Agents() {
  const { data: agents, isLoading } = useAgents()
  const sendMessage = useSendAgentMessage()
  const [chatAgent, setChatAgent] = useState<string | null>(null)
  const [chatInput, setChatInput] = useState('')
  const [chatResponse, setChatResponse] = useState('')

  const handleSend = async () => {
    if (!chatAgent || !chatInput.trim()) return
    try {
      const res = await sendMessage.mutateAsync({ name: chatAgent, message: chatInput })
      setChatResponse(res.content)
      setChatInput('')
    } catch (e: any) {
      setChatResponse(`Error: ${e.message}`)
    }
  }

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
        {agents?.map(agent => (
          <div key={agent.name} className="card agent-card">
            <div className="agent-header">
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <span className={`status-dot ${agent.online ? (agent.state === 'busy' ? 'busy' : 'online') : 'offline'}`} />
                <span className="agent-name">{agent.name}</span>
              </div>
              <span className={`badge ${agent.state === 'idle' ? 'badge-success' : agent.state === 'busy' ? 'badge-warning' : 'badge-idle'}`}>
                {agent.state}
              </span>
            </div>

            <div className="agent-type">{agent.type}</div>

            {agent.task && (
              <div style={{ fontSize: 13, color: 'var(--accent-gold)', fontWeight: 500 }}>
                ▶ {agent.task}
              </div>
            )}

            <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 4 }}>
              Status: {agent.online ? 'Online' : 'Offline'}
            </div>

            <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
              <button
                className="btn btn-ghost"
                style={{ fontSize: 12, padding: '6px 12px' }}
                onClick={() => setChatAgent(agent.name)}
              >
                Chat
              </button>
              <button className="btn btn-ghost" style={{ fontSize: 12, padding: '6px 12px' }}>
                Configure
              </button>
            </div>
          </div>
        ))}
        {isLoading && (
          <div className="card agent-card" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 100 }}>
            <div className="loading-spinner" />
          </div>
        )}
      </div>

      {/* Agent Chat Modal */}
      {chatAgent && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)',
          display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000,
        }} onClick={() => setChatAgent(null)}>
          <div className="card" style={{
            width: 500, maxHeight: '80vh', padding: 24, overflow: 'auto',
          }} onClick={e => e.stopPropagation()}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
              <h3 style={{ fontSize: 16, fontWeight: 600 }}>Chat with {chatAgent}</h3>
              <button className="btn btn-ghost" style={{ fontSize: 18 }} onClick={() => { setChatAgent(null); setChatResponse('') }}>
                ✕
              </button>
            </div>
            {chatResponse && (
              <div style={{
                background: 'var(--bg-secondary)', padding: 12, borderRadius: 8,
                marginBottom: 12, whiteSpace: 'pre-wrap', fontSize: 13,
              }}>
                {chatResponse}
              </div>
            )}
            <div style={{ display: 'flex', gap: 8 }}>
              <textarea
                className="chat-input"
                value={chatInput}
                onChange={e => setChatInput(e.target.value)}
                placeholder={`Message ${chatAgent}...`}
                onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }}
                rows={2}
                style={{
                  flex: 1, padding: '8px 12px', borderRadius: 8,
                  border: '1px solid var(--border-light)', fontFamily: 'Inter', fontSize: 13, resize: 'none',
                }}
              />
              <button className="btn btn-primary" onClick={handleSend} disabled={sendMessage.isPending}>
                {sendMessage.isPending ? '...' : 'Send'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
