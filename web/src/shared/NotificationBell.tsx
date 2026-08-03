import { useEffect, useRef, useState } from 'react'
import { Bell } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { getUnreadCount, listNotifications, markAllRead, markRead } from '../features/notifications/api'
import { describeNotification } from '../features/notifications/format'
import type { Notification } from '../features/notifications/types'
import { formatRelativeTime } from '../features/messaging/format'
import { ApiError } from '../api/client'
import { useUserDisplayName } from './useUserDisplayName'

const POLL_INTERVAL_MS = 30_000

// Appends "from <sender name>" to a notification's message when the
// payload carried a sender_id (consult/direct messages) — kept as its own
// component (rather than resolved inside describeNotification, a plain
// mapper) since name resolution needs a hook.
function NotificationMessage({ message, senderId }: { message: string; senderId?: string }) {
  const senderName = useUserDisplayName(senderId)
  if (!senderId) return <>{message}</>
  return (
    <>
      {message} from {senderName}
    </>
  )
}

export function NotificationBell() {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [unreadCount, setUnreadCount] = useState(0)
  const [notifications, setNotifications] = useState<Notification[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let cancelled = false
    function poll() {
      getUnreadCount()
        .then((res) => {
          if (!cancelled) setUnreadCount(res.count)
        })
        .catch(() => {
          // Silently ignore — badge just won't update this tick.
        })
    }
    poll()
    const id = setInterval(poll, POLL_INTERVAL_MS)
    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [])

  useEffect(() => {
    if (!open) return
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onClick)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  function loadNotifications() {
    setLoading(true)
    setError(null)
    listNotifications()
      .then((list) => setNotifications(list))
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Failed to load notifications'))
      .finally(() => setLoading(false))
  }

  function toggleOpen() {
    setOpen((v) => {
      const next = !v
      if (next) loadNotifications()
      return next
    })
  }

  function handleSelect(n: Notification) {
    const { to } = describeNotification(n)
    if (!n.read_at) {
      setNotifications((prev) => prev?.map((it) => (it.id === n.id ? { ...it, read_at: new Date().toISOString() } : it)) ?? prev)
      setUnreadCount((c) => Math.max(0, c - 1))
      markRead(n.id).catch(() => {
        // Best-effort — leave the optimistic UI as-is.
      })
    }
    setOpen(false)
    if (to) navigate(to)
  }

  function handleMarkAllRead() {
    setNotifications((prev) => prev?.map((it) => ({ ...it, read_at: it.read_at ?? new Date().toISOString() })) ?? prev)
    setUnreadCount(0)
    markAllRead().catch(() => {
      // Best-effort — badge/list already reflect the intended state.
    })
  }

  return (
    <div className="notification-bell" ref={ref}>
      <button
        type="button"
        className="notification-bell__trigger"
        onClick={toggleOpen}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="Notifications"
      >
        <Bell size={18} aria-hidden="true" />
        {unreadCount > 0 && (
          <span className="notification-bell__badge">{unreadCount > 99 ? '99+' : unreadCount}</span>
        )}
      </button>

      {open && (
        <div className="notification-bell__panel" role="menu">
          <div className="notification-bell__header">
            <span>Notifications</span>
            <button type="button" className="notification-bell__mark-all" onClick={handleMarkAllRead}>
              Mark all read
            </button>
          </div>

          {loading && <p className="notification-bell__status">Loading…</p>}
          {!loading && error && (
            <p className="notification-bell__status notification-bell__status--error">{error}</p>
          )}
          {!loading && !error && notifications && notifications.length === 0 && (
            <p className="notification-bell__status">No notifications yet</p>
          )}
          {!loading && !error && notifications && notifications.length > 0 && (
            <div className="notification-bell__list">
              {notifications.map((n) => {
                const { message, senderId } = describeNotification(n)
                return (
                  <button
                    type="button"
                    role="menuitem"
                    key={n.id}
                    className={`notification-bell__item${n.read_at ? '' : ' notification-bell__item--unread'}`}
                    onClick={() => handleSelect(n)}
                  >
                    <span className="notification-bell__item-message">
                      <NotificationMessage message={message} senderId={senderId} />
                    </span>
                    <span className="notification-bell__item-time">{formatRelativeTime(n.created_at)}</span>
                  </button>
                )
              })}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
