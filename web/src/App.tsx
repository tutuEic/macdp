import { Suspense, lazy } from 'react'
import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom'

// Lazy-load page components for code splitting
const Dashboard = lazy(() => import('./pages/Dashboard'))
const Projects = lazy(() => import('./pages/Projects'))
const Agents = lazy(() => import('./pages/Agents'))
const Kanban = lazy(() => import('./pages/Kanban'))
const Chat = lazy(() => import('./pages/Chat'))
import './App.css'

function PageLoader() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
      <div className="loading-spinner" />
    </div>
  )
}

function Sidebar() {
  const navItems = [
    { path: '/', label: 'Dashboard', icon: '◈' },
    { path: '/projects', label: 'Projects', icon: '◇' },
    { path: '/agents', label: 'Agents', icon: '◉' },
    { path: '/kanban', label: 'Kanban', icon: '▤' },
    { path: '/chat', label: 'Chat', icon: '◎' },
  ]

  return (
    <nav className="sidebar">
      <div className="sidebar-header">
        <div className="logo">
          <span className="logo-icon">◆</span>
          <span className="logo-text">MACDP</span>
        </div>
        <p className="logo-sub">Agent Console</p>
      </div>

      <div className="nav-section">
        <p className="nav-label">Navigation</p>
        {navItems.map(item => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
            end={item.path === '/'}
          >
            <span className="nav-icon">{item.icon}</span>
            <span className="nav-text">{item.label}</span>
          </NavLink>
        ))}
      </div>

      <div className="sidebar-footer">
        <div className="agent-status-mini">
          <span className="status-dot online"></span>
          <span>3 Agents Online</span>
        </div>
      </div>
    </nav>
  )
}

function App() {
  return (
    <BrowserRouter>
      <div className="app">
        <Sidebar />
        <main className="main-content">
          <Suspense fallback={<PageLoader />}>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/projects" element={<Projects />} />
              <Route path="/agents" element={<Agents />} />
              <Route path="/kanban" element={<Kanban />} />
              <Route path="/chat" element={<Chat />} />
            </Routes>
          </Suspense>
        </main>
      </div>
    </BrowserRouter>
  )
}

export default App
