import { useState, useEffect } from 'react'
import { useAgents, useChatHistory, useSendChatMessage, useProjects } from '../lib'

export default function Chat() {
  const { data: agents } = useAgents()
  const { data: projects } = useProjects()
  const [selectedProject, setSelectedProject] = useState('')
  const [activeAgent, setActiveAgent] = useState('')
  const [input, setInput] = useState('')
  const [messages, setMessages] = useState<any[]>([])

  const { data: history } = useChatHistory(selectedProject, activeAgent)
  const sendMessage = useSendChatMessage()

  // Load history when project/agent changes
  useEffect(() => {
    if (history) {
      // Reverse to show newest last
      setMessages([...history].reverse())
    }
  }, [history])

  const handleSend = async () => {
    if (!input.trim() || !activeAgent || !selectedProject) return
    const content = input
    setInput('')
    // Optimistic user message
    setMessages(prev => [...prev, { id: Date.now().toString(), role: 'user', content, agent_name: activeAgent }])
    try {
      const res = await sendMessage.mutateAsync({
        projectId: selectedProject,
        input: { agent_name: activeAgent, content },
      })
      setMessages(prev => [...prev, { id: res.id || Date.now().toString(), role: 'agent', content: res.content, agent_name: activeAgent }])
    } catch (e: any) {
      setMessages(prev => [...prev, { id: Date.now().toString(), role: 'system', content: `Error: ${e.message}`, agent_name: activeAgent }])
    }
  }

  // Auto-select first project and agent
  useEffect(() => {
    if (!selectedProject && projects?.length) setSelectedProject(projects[0].id)
  }, [projects, selectedProject])
  useEffect(() => {
    if (!activeAgent && agents?.length) {
      const online = agents.find(a => a.online)
      setActiveAgent(online?.name || agents[0]?.name || '')
    }
  }, [agents, activeAgent])

  return (
    <div className="page" style={{ padding: 0, height: 'calc(100vh - 60px)' }}>
      <div className="chat-container">
        {/* Agent list sidebar */}
        <div className="chat-sidebar">
          <h3 style={{ fontSize: 14, fontWeight: 600, marginBottom: 8, color: 'var(--text-primary)' }}>Project</h3>
          <select
            value={selectedProject}
            onChange={e => setSelectedProject(e.target.value)}
            style={{
              width: '100%', padding: '8px', borderRadius: 6, marginBottom: 16,
              border: '1px solid var(--border-light)', fontFamily: 'Inter', fontSize: 12,
            }}
          >
            <option value="">Select...</option>
            {projects?.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>

          <h3 style={{ fontSize: 14, fontWeight: 600, marginBottom: 8, color: 'var(--text-primary)' }}>Agents</h3>
          {agents?.map(agent => (
            <div
              key={agent.name}
              className={`chat-agent-item ${activeAgent === agent.name ? 'active' : ''}`}
              onClick={() => setActiveAgent(agent.name)}
            >
              <span className={`status-dot ${agent.online ? (agent.state === 'busy' ? 'busy' : 'online') : 'offline'}`} />
              <div>
                <div className="chat-agent-name">{agent.name}</div>
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{agent.state}</div>
              </div>
            </div>
          ))}
        </div>

        {/* Chat area */}
        <div className="chat-main">
          <div style={{
            padding: '16px 24px', borderBottom: '1px solid var(--border-card)',
            background: 'var(--bg-card)', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          }}>
            <div>
              <div style={{ fontSize: 16, fontWeight: 600 }}>{activeAgent || 'Select an agent'}</div>
              <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                {projects?.find(p => p.id === selectedProject)?.name || 'No project'}
              </div>
            </div>
            <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
              {messages.length} messages
            </div>
          </div>

          <div className="chat-messages">
            {messages.map(msg => (
              <div key={msg.id} className={`chat-message ${msg.role}`}>
                <div className="chat-bubble">{msg.content}</div>
                <div className="chat-time" style={{ textAlign: msg.role === 'user' ? 'right' : 'left' }}>
                  {msg.role === 'user' ? 'You' : msg.agent_name || msg.role}
                </div>
              </div>
            ))}
            {messages.length === 0 && (
              <div style={{ textAlign: 'center', color: 'var(--text-muted)', padding: 40 }}>
                {selectedProject && activeAgent
                  ? `Start a conversation with ${activeAgent}`
                  : 'Select a project and agent to start'}
              </div>
            )}
          </div>

          <div className="chat-input-area">
            <textarea
              className="chat-input"
              placeholder={activeAgent ? `Send a message to ${activeAgent}...` : 'Select an agent first...'}
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => {
                if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() }
              }}
              rows={1}
              disabled={!activeAgent || sendMessage.isPending}
            />
            <button
              className="btn btn-primary"
              onClick={handleSend}
              disabled={!activeAgent || !input.trim() || sendMessage.isPending}
            >
              {sendMessage.isPending ? '...' : 'Send'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
