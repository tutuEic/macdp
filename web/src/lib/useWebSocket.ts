import { useEffect, useRef, useState } from 'react'
import type { TaskEvent } from './types'

interface UseWebSocketOptions {
  onEvent?: (event: TaskEvent) => void
  onTaskStarted?: (event: TaskEvent) => void
  onTaskProgress?: (event: TaskEvent) => void
  onTaskCompleted?: (event: TaskEvent) => void
  onTaskFailed?: (event: TaskEvent) => void
  onAgentMessage?: (event: TaskEvent) => void
}

export function useWebSocket(opts: UseWebSocketOptions = {}) {
  const [connected, setConnected] = useState(false)
  const [lastEvent, setLastEvent] = useState<TaskEvent | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const optsRef = useRef(opts)
  optsRef.current = opts

  useEffect(() => {
    function connect() {
      const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
      const url = `${protocol}//${location.host}/ws`
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        setConnected(true)
      }

      ws.onmessage = (ev) => {
        try {
          const event: TaskEvent = JSON.parse(ev.data)
          setLastEvent(event)
          const o = optsRef.current
          o.onEvent?.(event)
          switch (event.type) {
            case 'task.started': o.onTaskStarted?.(event); break
            case 'task.progress': o.onTaskProgress?.(event); break
            case 'task.completed': o.onTaskCompleted?.(event); break
            case 'task.failed': o.onTaskFailed?.(event); break
            case 'agent.message': o.onAgentMessage?.(event); break
          }
        } catch { /* ignore parse errors */ }
      }

      ws.onclose = () => {
        setConnected(false)
        reconnectTimer.current = setTimeout(connect, 3000)
      }
    }

    connect()
    return () => {
      wsRef.current?.close()
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current)
    }
  }, [])

  return { connected, lastEvent }
}
