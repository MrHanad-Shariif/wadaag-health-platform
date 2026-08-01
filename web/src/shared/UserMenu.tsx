import { useEffect, useRef, useState } from 'react'
import { LogOut, Moon, Sun } from 'lucide-react'
import { useAuth } from '../features/auth/useAuth'
import { useTheme } from './useTheme'

function initials(name: string): string {
  const parts = name.replace(/@.*/, '').split(/[.\s_-]+/).filter(Boolean)
  const chars = parts.slice(0, 2).map((p) => p[0]?.toUpperCase() ?? '')
  return chars.join('') || name[0]?.toUpperCase() || '?'
}

export function UserMenu() {
  const { user, logout } = useAuth()
  const { theme, toggleTheme } = useTheme()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

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
        <span className="user-menu__avatar">{initials(name)}</span>
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
          <button type="button" role="menuitem" className="user-menu__item user-menu__item--danger" onClick={logout}>
            <LogOut size={16} aria-hidden="true" />
            Sign out
          </button>
        </div>
      )}
    </div>
  )
}
