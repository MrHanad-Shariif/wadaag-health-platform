import { useEffect, useRef, useState, type FormEvent } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { UserDisplayName } from '../../shared/UserDisplayName'
import { useUserDisplayName } from '../../shared/useUserDisplayName'
import { useAuth } from '../auth/useAuth'
import { listConversations, listMessages, markRead, sendMessage } from './api'
import { useConversationSocket } from './useConversationSocket'
import { otherPartyId, type Message } from './types'

// How often the REST fallback polls for new messages while the WebSocket
// is disconnected. The roadmap explicitly calls for "WebSocket with a
// polling fallback" — this is that fallback, not the primary transport.
const POLL_INTERVAL_MS = 7000
// How close to the bottom (px) counts as "scrolled to the end" for
// auto-mark-read purposes.
const BOTTOM_THRESHOLD_PX = 60
// How long a typing indicator stays up after the last "typing" event.
const TYPING_TIMEOUT_MS = 3000
// Minimum gap between outgoing "typing" WS events while the user types.
const TYPING_SEND_THROTTLE_MS = 1000

function mergeMessages(existing: Message[], incoming: Message[]): Message[] {
  const byId = new Map(existing.map((m) => [m.id, m]))
  for (const m of incoming) byId.set(m.id, m)
  return Array.from(byId.values()).sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
}

export function ConversationThreadPage() {
  const { conversationId } = useParams<{ conversationId: string }>()
  const { user } = useAuth()

  const historyState = useFetch(() => listMessages(conversationId!), [conversationId])
  // Best-effort label for who this conversation is with — there's no
  // single-conversation GET endpoint in the contract, so this reuses the
  // list endpoint and finds the matching row (see types.ts's
  // otherPartyLabel note on the underlying data gap).
  const conversationsState = useFetch(() => listConversations(), [])
  const conversation = conversationsState.data?.find((c) => c.id === conversationId)
  const resolvedOtherPartyName = useUserDisplayName(conversation ? otherPartyId(conversation, user?.id) : undefined)

  const [messages, setMessages] = useState<Message[]>([])
  const [typingUserId, setTypingUserId] = useState<string | null>(null)
  const [body, setBody] = useState('')
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState<string | null>(null)

  const containerRef = useRef<HTMLDivElement | null>(null)
  const typingTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const lastTypingSentRef = useRef(0)
  const lastMarkedIdRef = useRef<string | null>(null)

  useEffect(() => {
    if (historyState.data) {
      setMessages((prev) => mergeMessages(prev, historyState.data!))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [historyState.data])

  // Reset thread-local state when navigating between conversations.
  useEffect(() => {
    setMessages([])
    setTypingUserId(null)
    lastMarkedIdRef.current = null
  }, [conversationId])

  const { connected, sendTyping, sendRead } = useConversationSocket(conversationId, {
    onMessage: (message) => {
      setMessages((prev) => mergeMessages(prev, [message]))
    },
    onTyping: (userId) => {
      if (userId === user?.id) return
      setTypingUserId(userId)
      if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current)
      typingTimeoutRef.current = setTimeout(() => setTypingUserId(null), TYPING_TIMEOUT_MS)
    },
    onRead: () => {
      // Read receipts from the other party aren't surfaced in the UI yet
      // beyond marking our own side read — nothing to do here for now.
    },
  })

  // Polling fallback: only runs while the socket is down, so the thread
  // still updates if the WebSocket connection is unavailable/dropped.
  useEffect(() => {
    if (!conversationId || connected) return
    const interval = setInterval(() => {
      listMessages(conversationId)
        .then((latest) => setMessages((prev) => mergeMessages(prev, latest)))
        .catch(() => {
          // Transient poll failure — next tick will retry.
        })
    }, POLL_INTERVAL_MS)
    return () => clearInterval(interval)
  }, [conversationId, connected])

  function markReadIfAtBottom() {
    const container = containerRef.current
    if (!container || !conversationId || messages.length === 0) return
    const atBottom = container.scrollHeight - container.scrollTop - container.clientHeight < BOTTOM_THRESHOLD_PX
    if (!atBottom) return
    const latestId = messages[messages.length - 1].id
    if (lastMarkedIdRef.current === latestId) return
    lastMarkedIdRef.current = latestId
    markRead(conversationId).catch(() => {
      // Best-effort; the next scroll/message event will retry.
    })
    sendRead()
  }

  // Auto-scroll to the newest message and mark the thread read whenever
  // the message list grows (opening the thread counts as "reading" it).
  useEffect(() => {
    const container = containerRef.current
    if (container) container.scrollTop = container.scrollHeight
    markReadIfAtBottom()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [messages])

  useEffect(() => {
    return () => {
      if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current)
    }
  }, [])

  function handleBodyChange(value: string) {
    setBody(value)
    const now = Date.now()
    if (now - lastTypingSentRef.current > TYPING_SEND_THROTTLE_MS) {
      sendTyping()
      lastTypingSentRef.current = now
    }
  }

  async function handleSend(e: FormEvent) {
    e.preventDefault()
    const trimmed = body.trim()
    if (!trimmed || !conversationId) return
    setSendError(null)
    setSending(true)
    try {
      const message = await sendMessage(conversationId, trimmed)
      setMessages((prev) => mergeMessages(prev, [message]))
      setBody('')
    } catch (err) {
      setSendError(err instanceof ApiError ? err.message : 'Could not send message.')
    } finally {
      setSending(false)
    }
  }

  if (historyState.loading && messages.length === 0) return <LoadingState />
  if (historyState.error) return <ErrorState message={historyState.error} />

  const title = conversation ? resolvedOtherPartyName : 'Conversation'

  return (
    <div className="page">
      <PageHeader
        breadcrumb={[{ label: 'Messages', to: '/messages' }, { label: title }]}
        title={title}
        actions={
          <span className={`connection-indicator${connected ? ' connection-indicator--live' : ''}`}>
            <span className="connection-indicator__dot" aria-hidden="true" />
            {connected ? 'Live' : 'Reconnecting…'}
          </span>
        }
      />

      <div className="thread-scroll" ref={containerRef} onScroll={markReadIfAtBottom}>
        <div className="consult-thread">
          {messages.length === 0 && <p className="empty-state">No messages yet — say hello.</p>}
          {messages.map((m) => (
            <div
              key={m.id}
              className={`consult-thread__message${m.sender_id === user?.id ? ' consult-thread__message--mine' : ''}`}
            >
              <div className="consult-thread__meta">
                <span>{m.sender_id === user?.id ? 'You' : <UserDisplayName userId={m.sender_id} />}</span>
                <span>{new Date(m.created_at).toLocaleString()}</span>
              </div>
              <div className="consult-thread__body">{m.body}</div>
            </div>
          ))}
        </div>
      </div>

      {typingUserId && <p className="typing-indicator">Typing…</p>}

      <form className="form" onSubmit={handleSend}>
        <label htmlFor="messageBody">Message</label>
        <textarea
          id="messageBody"
          value={body}
          onChange={(e) => handleBodyChange(e.target.value)}
          rows={3}
          required
        />
        {sendError && <p role="alert" className="form-error">{sendError}</p>}
        <button type="submit" disabled={sending || !body.trim()}>
          {sending ? 'Sending…' : 'Send'}
        </button>
      </form>
    </div>
  )
}
