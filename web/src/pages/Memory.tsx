import { useState } from 'react'
import { useProjects, useMemoryStats } from '../lib'

export default function MemoryPanel() {
  const { data: projects } = useProjects()
  const [selectedProject, setSelectedProject] = useState('')
  const { data: stats, isLoading } = useMemoryStats(selectedProject)

  const project = projects?.find(p => p.id === selectedProject)

  // Get real values from API or show defaults
  const totalEntries = stats?.total_entries ?? 0
  const totalTokens = stats?.total_tokens ?? 0
  const workingCount = stats?.by_tier?.working ?? 0
  const shortCount = stats?.by_tier?.short ?? 0
  const longCount = stats?.by_tier?.long ?? 0
  const summaryCount = stats?.by_category?.summary ?? 0
  const decisionCount = stats?.by_category?.decision ?? 0
  const conventionCount = stats?.by_category?.convention ?? 0

  const tiers = [
    { tier: 'Working', count: workingCount, desc: 'Dependency outputs + decisions', limit: '≤2K tokens', color: 'var(--success)' },
    { tier: 'Short-term', count: shortCount, desc: 'Task summaries + file changes', limit: '≤3K tokens', color: 'var(--accent-gold)' },
    { tier: 'Long-term', count: longCount, desc: 'Decisions + conventions + patterns', limit: 'unlimited', color: 'var(--info)' },
  ]

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Memory</h1>
          <p className="page-subtitle">Agent memory management and token budget</p>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px' }}>
        {/* Stats */}
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

          {isLoading && (
            <div style={{ display: 'flex', justifyContent: 'center', padding: 20 }}>
              <div className="loading-spinner" />
            </div>
          )}

          {project && !isLoading && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <div style={{ fontSize: 13, color: 'var(--text-muted)' }}>
                Project: <strong style={{ color: 'var(--text-primary)' }}>{project.name}</strong>
              </div>

              {/* Topline stats */}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8 }}>
                <div className="card" style={{ padding: 12, textAlign: 'center', background: 'var(--bg-secondary)' }}>
                  <div style={{ fontSize: 20, fontWeight: 700, color: 'var(--accent-gold)' }}>{totalEntries}</div>
                  <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>Entries</div>
                </div>
                <div className="card" style={{ padding: 12, textAlign: 'center', background: 'var(--bg-secondary)' }}>
                  <div style={{ fontSize: 20, fontWeight: 700, color: 'var(--info)' }}>{totalTokens}</div>
                  <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>Total Tokens</div>
                </div>
                <div className="card" style={{ padding: 12, textAlign: 'center', background: 'var(--bg-secondary)' }}>
                  <div style={{ fontSize: 20, fontWeight: 700, color: 'var(--success)' }}>{summaryCount}</div>
                  <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>Summaries</div>
                </div>
              </div>

              {/* Memory tiers */}
              <div style={{ marginTop: 8 }}>
                <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8, color: 'var(--text-secondary)' }}>
                  Memory Tiers
                </div>
                {tiers.map(t => (
                  <div key={t.tier} style={{
                    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                    padding: '10px 14px', background: 'var(--bg-secondary)',
                    borderRadius: 6, marginBottom: 6,
                  }}>
                    <div>
                      <div style={{ fontSize: 13, fontWeight: 500, color: t.color }}>{t.tier}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{t.desc}</div>
                    </div>
                    <div style={{ textAlign: 'right' }}>
                      <div style={{ fontSize: 16, fontWeight: 700, color: t.color }}>{t.count}</div>
                      <span className="badge badge-idle" style={{ fontSize: 10 }}>{t.limit}</span>
                    </div>
                  </div>
                ))}
              </div>

              {/* Decision count */}
              {decisionCount > 0 && (
                <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 4 }}>
                  📋 {decisionCount} architectural decisions · 📐 {conventionCount} conventions
                </div>
              )}
            </div>
          )}

          {!project && (
            <div style={{ color: 'var(--text-muted)', padding: 20, textAlign: 'center', fontSize: 13 }}>
              Select a project to view memory stats
            </div>
          )}
        </div>

        {/* Techniques */}
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
