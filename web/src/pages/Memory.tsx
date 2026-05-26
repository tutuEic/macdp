import { useState } from 'react'
import { useProjects } from '../lib'

// Memory panel component - shows memory stats and entries for a project
// Uses the existing events and projects API since memory stats aren't exposed via REST yet

export default function MemoryPanel() {
  const { data: projects } = useProjects()
  const [selectedProject, setSelectedProject] = useState('')

  const project = projects?.find(p => p.id === selectedProject)

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Memory</h1>
          <p className="page-subtitle">Agent memory management and token budget</p>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px' }}>
        {/* Project selector + stats */}
        <div className="card" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: 16, marginBottom: 16, fontWeight: 600 }}>Project Memory</h3>

          <select
            value={selectedProject}
            onChange={e => setSelectedProject(e.target.value)}
            style={{
              width: '100%', padding: '10px 14px', borderRadius: 8, marginBottom: 20,
              border: '1px solid var(--border-light)', fontFamily: 'Inter', fontSize: 13,
              background: 'var(--bg-card)',
            }}
          >
            <option value="">Select a project...</option>
            {projects?.map(p => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>

          {project && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <div style={{ fontSize: 13, color: 'var(--text-muted)' }}>
                Project: <strong style={{ color: 'var(--text-primary)' }}>{project.name}</strong>
              </div>

              {/* Memory tiers visual */}
              <div style={{ marginTop: 8 }}>
                <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8, color: 'var(--text-secondary)' }}>
                  Memory Tiers
                </div>
                {[
                  { tier: 'Working', desc: 'Dependency outputs + decisions', limit: '≤2K tokens', color: 'var(--success)' },
                  { tier: 'Short-term', desc: 'Task summaries + file changes', limit: '≤3K tokens', color: 'var(--accent-gold)' },
                  { tier: 'Long-term', desc: 'Decisions + conventions + patterns', limit: 'unlimited', color: 'var(--info)' },
                ].map(t => (
                  <div key={t.tier} style={{
                    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                    padding: '10px 14px', background: 'var(--bg-secondary)',
                    borderRadius: 6, marginBottom: 6,
                  }}>
                    <div>
                      <div style={{ fontSize: 13, fontWeight: 500, color: t.color }}>{t.tier}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{t.desc}</div>
                    </div>
                    <span className="badge badge-idle" style={{ fontSize: 11 }}>{t.limit}</span>
                  </div>
                ))}
              </div>

              {/* Token budget */}
              <div style={{ marginTop: 8 }}>
                <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8, color: 'var(--text-secondary)' }}>
                  Token Budget (per agent call)
                </div>
                <div className="progress-bar" style={{ height: 8 }}>
                  <div className="progress-bar-fill" style={{ width: '25%', background: 'var(--success)' }} />
                  <div className="progress-bar-fill" style={{ width: '38%', background: 'var(--accent-gold)', marginLeft: '25%', position: 'absolute' }} />
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>
                  <span>Working: ~2000</span>
                  <span>Retrieved: ~3000</span>
                  <span>Max: 8000</span>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Memory techniques info */}
        <div className="card" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: 16, marginBottom: 16, fontWeight: 600 }}>Token-saving Techniques</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            {[
              { icon: '📦', title: 'Auto-Summarization', desc: 'Task output 10K→300 tokens after completion' },
              { icon: '🔄', title: 'Hierarchical Compression', desc: 'Task → Module → Project summaries' },
              { icon: '🎯', title: 'Relevance Retrieval', desc: 'Only top-K most similar past summaries injected' },
              { icon: '📊', title: 'Token Budget', desc: 'Per-call 8K cap with breakdown by tier' },
              { icon: '🧠', title: 'Decision Extraction', desc: 'Auto-detect architectural choices' },
              { icon: '🧹', title: 'Auto-Pruning', desc: 'Keep last 20 entries per category' },
            ].map((t, i) => (
              <div key={i} style={{
                display: 'flex', gap: 12, alignItems: 'flex-start',
                padding: '10px 14px', background: 'var(--bg-secondary)', borderRadius: 6,
              }}>
                <span style={{ fontSize: 18 }}>{t.icon}</span>
                <div>
                  <div style={{ fontSize: 13, fontWeight: 500 }}>{t.title}</div>
                  <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{t.desc}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
