import { LayoutDashboard, LogOut, ClipboardList, Moon, Sun, UserPlus } from 'lucide-react'
import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../features/auth/useAuth'
import { useTheme } from './useTheme'
import { BrandMark } from './BrandMark'

export function Layout() {
  const { user, logout } = useAuth()
  const { theme, toggleTheme } = useTheme()
  const isProvider = user?.role === 'physician' || user?.role === 'hospital_admin'

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <BrandMark />
          <span>Wadaag Health</span>
        </div>

        <nav className="sidebar-nav">
          <NavLink to="/" end className="sidebar-link">
            <LayoutDashboard size={18} aria-hidden="true" />
            <span>Dashboard</span>
          </NavLink>
          {isProvider && (
            <>
              <NavLink to="/patients/new" className="sidebar-link">
                <UserPlus size={18} aria-hidden="true" />
                <span>New patient</span>
              </NavLink>
              <NavLink to="/referrals" className="sidebar-link">
                <ClipboardList size={18} aria-hidden="true" />
                <span>Referrals</span>
              </NavLink>
            </>
          )}
        </nav>

        <div className="sidebar-user">
          <div className="sidebar-user__identity">
            <span className="sidebar-user__name">{user?.email ?? user?.phone}</span>
            <span className="sidebar-user__role">{user?.role.replace('_', ' ')}</span>
          </div>
          <button
            type="button"
            className="icon-button"
            onClick={toggleTheme}
            aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
          >
            {theme === 'dark' ? <Sun size={18} aria-hidden="true" /> : <Moon size={18} aria-hidden="true" />}
          </button>
          <button type="button" className="icon-button" onClick={logout} aria-label="Sign out">
            <LogOut size={18} aria-hidden="true" />
          </button>
        </div>
      </aside>
      <main className="app-content">
        <Outlet />
      </main>
    </div>
  )
}
