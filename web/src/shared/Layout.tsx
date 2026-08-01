import { useState, type ComponentType } from 'react'
import {
  ChevronsLeft,
  ChevronsRight,
  ClipboardList,
  LayoutDashboard,
  ShieldCheck,
  UserCog,
  UserPlus,
  Users,
} from 'lucide-react'
import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../features/auth/useAuth'
import { hasPermission } from '../features/auth/permissions'
import { UserMenu } from './UserMenu'
import { BrandMark } from './BrandMark'

interface NavItem {
  to: string
  label: string
  icon: ComponentType<{ size?: number; 'aria-hidden'?: boolean }>
  end?: boolean
}

interface NavGroup {
  label: string
  items: NavItem[]
}

export function Layout() {
  const { user } = useAuth()
  const isProvider = user?.role === 'physician' || user?.role === 'hospital_admin'
  const isSystemAdmin = user?.role === 'system_admin'
  const canSeeAuthentication = hasPermission(user, 'users', 'read') || hasPermission(user, 'roles', 'read')

  const [collapsed, setCollapsed] = useState(() => localStorage.getItem('sidebar_collapsed') === 'true')

  function toggleCollapsed() {
    setCollapsed((prev) => {
      const next = !prev
      localStorage.setItem('sidebar_collapsed', String(next))
      return next
    })
  }

  const groups: NavGroup[] = [
    { label: 'Workspace', items: [{ to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true }] },
  ]
  if (isProvider) {
    groups.push({
      label: 'Care',
      items: [
        { to: '/patients/new', label: 'New patient', icon: UserPlus },
        { to: '/referrals', label: 'Referrals', icon: ClipboardList },
      ],
    })
  }
  if (isSystemAdmin) {
    groups.push({ label: 'Records', items: [{ to: '/patients', label: 'Patients', icon: Users }] })
  }
  if (canSeeAuthentication) {
    groups.push({
      label: 'Administration',
      items: [
        { to: '/authentication/users', label: 'Users', icon: UserCog },
        { to: '/authentication/roles', label: 'Roles', icon: ShieldCheck },
      ],
    })
  }

  return (
    <div className="app-shell">
      <aside className={`sidebar${collapsed ? ' sidebar--collapsed' : ''}`}>
        <div className="sidebar-brand">
          <BrandMark />
          <span>Wadaag Health</span>
        </div>

        <nav className="sidebar-nav">
          {groups.map((group) => (
            <div className="sidebar-group" key={group.label}>
              <span className="sidebar-group__label">{group.label}</span>
              {group.items.map((item) => (
                <NavLink key={item.to} to={item.to} end={item.end} className="sidebar-link" title={item.label}>
                  <item.icon size={18} aria-hidden={true} />
                  <span>{item.label}</span>
                </NavLink>
              ))}
            </div>
          ))}
        </nav>

        <button
          type="button"
          className="sidebar-collapse-toggle"
          onClick={toggleCollapsed}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {collapsed ? <ChevronsRight size={16} aria-hidden="true" /> : <ChevronsLeft size={16} aria-hidden="true" />}
          <span>Collapse</span>
        </button>
      </aside>

      <div className="app-main">
        <header className="topbar">
          <span className="topbar__brand">
            <BrandMark size={20} />
            <span>Wadaag Health</span>
          </span>
          <div className="topbar__spacer" />
          <UserMenu />
        </header>
        <main className="app-content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
