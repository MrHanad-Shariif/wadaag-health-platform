import { useEffect, useRef, useState } from 'react'
import { API_URL } from '../../api/client'
import type { ClientSocketEvent, Message, ServerSocketEvent } from './types'

interface ConversationSocketHandlers {
  onMessage?: (message: Message) => void
  onTyping?: (userId: string) => void
  onRead?: (userId: string, readAt: string) => void
}

const RECONNECT_DELAY_MS = 3000

function buildSocketUrl(conversationId: string): string | null {
  const token = localStorage.getItem('access_token')
  if (!token) return null
  const wsBase = API_URL.replace(/^http/, 'ws')
  return `${wsBase}/api/v1/conversations/${conversationId}/ws?token=${encodeURIComponent(token)}`
}

// Wraps the conversation WebSocket: connects, dispatches parsed server
// events to the given handlers, and reconnects on a simple fixed delay if
// the socket drops. Sending a chat message always goes through the REST
// POST — this hook is receive-only push plus the two ephemeral
// typing/read signals sent back to the server.
//
// Callers should keep using the REST endpoints (listMessages/markRead) as
// a polling fallback — `connected` reflects live socket status so a page
// can decide whether to poll more aggressively while it's false.
export function useConversationSocket(conversationId: string | undefined, handlers: ConversationSocketHandlers) {
  const [connected, setConnected] = useState(false)
  const socketRef = useRef<WebSocket | null>(null)
  const handlersRef = useRef(handlers)
  handlersRef.current = handlers

  useEffect(() => {
    if (!conversationId) return

    let cancelled = false
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined

    function connect() {
      if (cancelled || !conversationId) return
      const url = buildSocketUrl(conversationId)
      if (!url) return

      const socket = new WebSocket(url)
      socketRef.current = socket

      socket.onopen = () => {
        if (cancelled) return
        setConnected(true)
      }

      socket.onmessage = (event) => {
        let parsed: ServerSocketEvent
        try {
          parsed = JSON.parse(event.data)
        } catch {
          return
        }
        switch (parsed.type) {
          case 'message':
            handlersRef.current.onMessage?.(parsed.message)
            break
          case 'typing':
            handlersRef.current.onTyping?.(parsed.user_id)
            break
          case 'read':
            handlersRef.current.onRead?.(parsed.user_id, parsed.read_at)
            break
        }
      }

      socket.onclose = () => {
        socketRef.current = null
        if (cancelled) return
        setConnected(false)
        reconnectTimer = setTimeout(connect, RECONNECT_DELAY_MS)
      }

      socket.onerror = () => {
        socket.close()
      }
    }

    connect()

    return () => {
      cancelled = true
      if (reconnectTimer) clearTimeout(reconnectTimer)
      socketRef.current?.close()
      socketRef.current = null
      setConnected(false)
    }
  }, [conversationId])

  function send(evt: ClientSocketEvent) {
    const socket = socketRef.current
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(evt))
    }
  }

  return {
    connected,
    sendTyping: () => send({ type: 'typing' }),
    sendRead: () => send({ type: 'read' }),
  }
}
