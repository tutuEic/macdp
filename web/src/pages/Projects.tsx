import { useState } from 'react'
import { useProjects, useCreateProject, usePlanProject, useExecuteProject } from '../lib'

export default function Projects() {
  const { data: projects, isLoading } = useProjects()
  const createProject = useCreateProject()
  const planProject = usePlanProject()
  const executeProject = useExecuteProject()

  const [showNew, setShowNew] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [repoPath, setRepoPath] = useState('')
  const [planReq, setPlanReq] = useState('')
  const [planningId, setPlanningId] = useState<string | null>(null)
  const [planResult, setPlanResult] = useState<string>('')

  const handleCreate = async () => {
    if (!name.trim()) return
    try {
      await createProject.mutateAsync({ name, description, repo_path: repoPath })
      setName(''); setDescription(''); setRepoPath(''); setShowNew(false)
    } catch (e: any) {
      alert(`Failed: ${e.message}`)
    }
  }

  const handlePlan = async (id: string) => {
    if (!planReq.trim()) return
    try {
      const result = await planProject.mutateAsync({ id, input: { requirement: planReq } })
      setPlanResult(JSON.stringify(result, null, 2))
    } catch (e: any) {
      setPlanResult(`Error: ${e.message}`)
    }
  }

  const handleExecute = async (id: string) => {
    try {
      await executeProject.mutateAsync(id)
    } catch (e: any) {
      alert(`Execute failed: ${e.message}`)
    }
  }

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
        <div className="card animate-fade-in" style={{ padding: 24, marginBottom: 24 }}>
          <h3 style={{ fontSize: 16, marginBottom: 16, fontFamily: 'Inter', fontWeight: 600 }}>Create Project</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <input className="input" placeholder="Project name" value={name} onChange={e => setName(e.target.value)} />
            <textarea className="input" placeholder="Description" rows={3} style={{ resize: 'vertical' }}
              value={description} onChange={e => setDescription(e.target.value)} />
            <input className="input" placeholder="Repository path (e.g., /mnt/d/projects/myapp)"
              value={repoPath} onChange={e => setRepoPath(e.target.value)} />
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button className="btn btn-secondary" onClick={() => setShowNew(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={handleCreate}
                disabled={createProject.isPending}>
                {createProject.isPending ? 'Creating...' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      {isLoading && (
        <div style={{ display: 'flex', justifyContent: 'center', padding: 40 }}>
          <div className="loading-spinner" />
        </div>
      )}

      <div className="project-grid">
        {projects?.map(p => (
          <div key={p.id} className="card project-card">
            <div>
              <div className="project-name">{p.name}</div>
              <div className="project-desc">{p.description}</div>
              {p.repo_path && (
                <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>
                  📁 {p.repo_path}
                </div>
              )}
            </div>

            <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 8 }}>
              Created: {new Date(p.created_at).toLocaleDateString()}
            </div>

            <div style={{ display: 'flex', gap: 8, marginTop: 12, flexWrap: 'wrap' }}>
              <button className="btn btn-ghost" style={{ fontSize: 12, padding: '6px 12px' }}
                onClick={() => { setPlanningId(p.id); setPlanReq(''); setPlanResult('') }}>
                Plan
              </button>
              <button className="btn btn-primary" style={{ fontSize: 12, padding: '6px 12px' }}
                onClick={() => handleExecute(p.id)}
                disabled={executeProject.isPending}>
                {executeProject.isPending ? '...' : 'Execute'}
              </button>
            </div>

            {/* Inline plan form */}
            {planningId === p.id && (
              <div style={{ marginTop: 12, padding: 12, background: 'var(--bg-secondary)', borderRadius: 8 }}>
                <textarea
                  className="input"
                  placeholder="Describe what you want to build..."
                  rows={3}
                  style={{ resize: 'vertical', fontSize: 13, marginBottom: 8 }}
                  value={planReq}
                  onChange={e => setPlanReq(e.target.value)}
                />
                <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
                  <button className="btn btn-ghost" style={{ fontSize: 11 }} onClick={() => setPlanningId(null)}>Cancel</button>
                  <button className="btn btn-primary" style={{ fontSize: 11 }}
                    onClick={() => handlePlan(p.id)}
                    disabled={planProject.isPending}>
                    {planProject.isPending ? 'Planning...' : 'Generate Plan'}
                  </button>
                </div>
                {planResult && (
                  <pre style={{
                    marginTop: 8, padding: 12, background: 'var(--bg-card)',
                    borderRadius: 6, fontSize: 11, overflow: 'auto', maxHeight: 200,
                  }}>{planResult}</pre>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
