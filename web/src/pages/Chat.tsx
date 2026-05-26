import { useState } from 'react'

interface Message {
  id: string
  role: 'user' | 'agent' | 'system'
  agent: string
  content: string
  time: string
}

export default function Chat() {
  const [activeAgent, setActiveAgent] = useState('Claude Code')
  const [input, setInput] = useState('')

  const agents = [
    { name: 'Hermes', online: true, state: 'idle' },
    { name: 'Claude Code', online: true, state: 'busy' },
    { name: 'Codex', online: true, state: 'idle' },
    { name: 'OpenClaw', online: false, state: 'offline' },
  ]

  const [messages] = useState<Message[]>([
    { id: '1', role: 'system', agent: 'Claude Code', content: 'Task T-003 "Login API" assigned to Claude Code', time: '14:30' },
    { id: '2', role: 'user', agent: 'Claude Code', content: 'Implement user login API with JWT support. Use Pydantic v2 for request validation.', time: '14:30' },
    { id: '3', role: 'agent', agent: 'Claude Code', content: 'I will implement the login API. Starting with the Pydantic models and endpoint.', time: '14:31' },
    { id: '4', role: 'agent', agent: 'Claude Code', content: 'Created src/models/auth.py with LoginRequest and TokenResponse models. Created src/api/auth.py with POST /auth/login endpoint that validates credentials and returns JWT tokens.', time: '14:32' },
    { id: '5', role: 'agent', agent: 'Claude Code', content: 'Task complete. Changed files: src/models/auth.py, src/api/auth.py. Committed to feature/T-003 branch.', time: '14:33' },
  ])

  const filteredMessages = messages.filter(m => m.agent === activeAgent)

  const handleSend = () => {
    if (!input.trim()) return
    // TODO: send to backend
    setInput('')
  }

  return (
    <div className="page" style={{padding:0, height:'calc(100vh)'}}>
      <div className="chat-container">
        {/* Agent list */}
        <div className="chat-sidebar">
          <h3 style={{fontSize:14, fontWeight:600, marginBottom:16, color:'var(--text-primary)'}}>Agents</h3>
          {agents.map(agent => (
            <div
              key={agent.name}
              className={`chat-agent-item ${activeAgent === agent.name ? 'active' : ''}`}
              onClick={() => setActiveAgent(agent.name)}
            >
              <span className={`status-dot ${agent.online ? (agent.state === 'busy' ? 'busy' : 'online') : 'offline'}`}></span>
              <div>
                <div className="chat-agent-name">{agent.name}</div>
                <div style={{fontSize:11, color:'var(--text-muted)'}}>{agent.state}</div>
              </div>
            </div>
          ))}
        </div>

        {/* Chat area */}
        <div className="chat-main">
          <div style={{padding:'16px 24px', borderBottom:'1px solid var(--border-card)', background:'var(--bg-card)', display:'flex', alignItems:'center', justifyContent:'space-between'}}>
            <div>
              <div style={{fontSize:16, fontWeight:600}}>{activeAgent}</div>
              <div style={{fontSize:12, color:'var(--text-muted)'}}>Task T-003: Login API</div>
            </div>
            <div style={{display:'flex', gap:8}}>
              <button className="btn btn-ghost" style={{fontSize:12}}>Pause</button>
              <button className="btn btn-ghost" style={{fontSize:12, color:'var(--error)'}}>Cancel</button>
            </div>
          </div>

          <div className="chat-messages">
            {filteredMessages.map(msg => (
              <div key={msg.id} className={`chat-message ${msg.role}`}>
                <div className="chat-bubble">{msg.content}</div>
                <div className="chat-time" style={{textAlign: msg.role === 'user' ? 'right' : 'left'}}>{msg.time}</div>
              </div>
            ))}
          </div>

          <div className="chat-input-area">
            <textarea
              className="chat-input"
              placeholder={`Send a message to ${activeAgent}...`}
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }}
              rows={1}
            />
            <button className="btn btn-primary" onClick={handleSend}>Send</button>
          </div>
        </div>
      </div>
    </div>
  )
}
