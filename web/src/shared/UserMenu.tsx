import { useEffect, useRef, useState } from 'react'
import { LogOut, Monitor, Moon, Stethoscope, Sun, UserCircle } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../features/auth/useAuth'
import { getMyProfile } from '../features/profile/api'
import { useTheme } from './useTheme'
import { initials } from './avatar'

export function UserMenu() {
  const { user, logout } = useAuth()
  const { theme, toggleTheme } = useTheme()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  // Optimistic default: a successful login sets is_online=true server-side, so
  // assume online on mount and reconcile with the real value once it loads —
  // avoids a flash of "offline" while the request is in flight.
  const [isOnline, setIsOnline] = useState(true)

  useEffect(() => {
    let cancelled = false
    getMyProfile()
      .then((profile) => {
        if (!cancelled) setIsOnline(profile.is_online)
      })
      .catch(() => {
        // Keep the optimistic default if the profile fetch fails.
      })
    return () => {
      cancelled = true
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

  if (!user) return null
  const name = user.email ?? user.phone ?? 'Account'

  return (
    <div className="user-menu" ref={ref}>
      <button
        type="button"
        className="user-menu__trigger"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <span className="user-menu__avatar">
          {initials(name)}
          <span
            className={`user-menu__presence-dot user-menu__presence-dot--${isOnline ? 'online' : 'offline'}`}
            aria-label={isOnline ? 'Online' : 'Offline'}
            title={isOnline ? 'Online' : 'Offline'}
          />
        </span>
        <span className="user-menu__identity">
          <span className="user-menu__name">{name}</span>
          <span className="user-menu__role">{user.role.replace('_', ' ')}</span>
        </span>
      </button>

      {open && (
        <div className="user-menu__panel" role="menu">
          <button type="button" role="menuitem" className="user-menu__item" onClick={toggleTheme}>
            {theme === 'dark' ? <Sun size={16} aria-hidden="true" /> : <Moon size={16} aria-hidden="true" />}
            {theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
          </button>
          <button
            type="button"
            role="menuitem"
            className="user-menu__item"
            onClick={() => {
              setOpen(false)
              navigate('/profile')
            }}
          >
            <UserCircle size={16} aria-hidden="true" />
            Edit profile
          </button>
          {user.role === 'physician' && (
            <button
              type="button"
              role="menuitem"
              className="user-menu__item"
              onClick={() => {
                setOpen(false)
                navigate('/providers/me')
              }}
            >
              <Stethoscope size={16} aria-hidden="true" />
              My provider profile
            </button>
          )}
          {/* Phase 0 stub — move under Settings once Phase 9 lands. */}
          <button
            type="button"
            role="menuitem"
            className="user-menu__item"
            onClick={() => {
              setOpen(false)
              navigate('/settings/sessions')
            }}
          >
            <Monitor size={16} aria-hidden="true" />
            Active sessions
          </button>
          <button type="button" role="menuitem" className="user-menu__item user-menu__item--danger" onClick={logout}>
            <LogOut size={16} aria-hidden="true" />
            Sign out
          </button>
        </div>
      )}
    </div>
  )
}
