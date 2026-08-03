import { useState, type ComponentType, type FormEvent } from 'react'
import {
  Building2,
  CalendarClock,
  CalendarDays,
  ChevronsLeft,
  ChevronsRight,
  ClipboardList,
  Flag,
  History,
  LayoutDashboard,
  MessageSquare,
  Search,
  ShieldCheck,
  Stethoscope,
  UserCog,
  UserPlus,
  Users,
} from 'lucide-react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../features/auth/useAuth'
import { hasPermission } from '../features/auth/permissions'
import { UserMenu } from './UserMenu'
import { NotificationBell } from './NotificationBell'
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

// A simple, non-live search box: it just navigates to /search?q=... on
// submit rather than showing a live-results dropdown as-you-type.
function TopbarSearch() {
  const navigate = useNavigate()
  const [value, setValue] = useState('')

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const q = value.trim()
    if (!q) return
    navigate(`/search?q=${encodeURIComponent(q)}`)
  }

  return (
    <form className="topbar-search" role="search" onSubmit={handleSubmit}>
      <Search size={16} aria-hidden="true" />
      <input
        type="search"
        placeholder="Search patients, doctors, hospitals…"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        aria-label="Search"
      />
    </form>
  )
}

export function Layout() {
  const { user } = useAuth()
  const isProvider = user?.role === 'physician' || user?.role === 'hospital_admin'
  const isPatient = user?.role === 'patient'
  const isSystemAdmin = user?.role === 'system_admin'
  const canManageFacilities = isSystemAdmin || user?.role === 'hospital_admin'
  const canSeeAuthentication = hasPermission(user, 'users', 'read') || hasPermission(user, 'roles', 'read')

  const [collapsed, setCollapsed] = useState(() => localStorage.getItem('sidebar_collapsed') === 'true')

  function toggleCollapsed() {
    setCollapsed((prev) => {
      const next = !prev
      localStorage.setItem('sidebar_collapsed', String(next))
      return next
    })
  }

  const workspaceItems: NavItem[] = [{ to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true }]
  if (isProvider || isPatient) {
    // Provider-side (own schedule) and patient-side (own bookings) both
    // land on the same "/appointments" list — which of those it shows is
    // role-aware server-side (GET /appointments/mine).
    workspaceItems.push({ to: '/appointments', label: 'Appointments', icon: CalendarDays })
  }
  // Visible to every authenticated role, patients included — this is
  // general communication, not clinician-only, so it lives here rather
  // than in one of the role-gated groups below.
  workspaceItems.push({ to: '/messages', label: 'Messages', icon: MessageSquare })

  const groups: NavGroup[] = [
    {
      label: 'Workspace',
      items: workspaceItems,
    },
  ]
  if (isProvider) {
    const careItems: NavItem[] = [
      { to: '/patients/new', label: 'New patient', icon: UserPlus },
      { to: '/referrals', label: 'Referrals', icon: ClipboardList },
      { to: '/consults', label: 'Consultations', icon: Stethoscope },
    ]
    // Availability rules are owned by a provider profile — only physicians
    // have one (see ProviderProfilePage/UserMenu's own "physician"-only
    // gate on "/providers/me"), so hospital_admin doesn't get this link.
    if (user?.role === 'physician') {
      careItems.push({ to: '/appointments/availability', label: 'My availability', icon: CalendarClock })
    }
    groups.push({
      label: 'Care',
      items: careItems,
    })
  }
  if (isSystemAdmin) {
    groups.push({
      label: 'Records',
      items: [
        { to: '/patients', label: 'Patients', icon: Users },
        { to: '/patients/new', label: 'New patient', icon: UserPlus },
        // system_admin gets full oversight visibility into referrals and
        // consultations (backend serves an unscoped list for this role —
        // see referrals/consults Handler.inbox), but not the ability to
        // act as a clinician: accept/decline/reply/close actions on those
        // detail pages are still gated to physician/hospital_admin there.
        { to: '/referrals', label: 'Referrals', icon: ClipboardList },
        { to: '/consults', label: 'Consultations', icon: Stethoscope },
      ],
    })
  }
  if (canManageFacilities) {
    groups.push({
      label: 'Facilities',
      items: [{ to: '/facilities', label: 'Facilities', icon: Building2 }],
    })
  }
  if (canSeeAuthentication) {
    groups.push({
      label: 'AUTHENTICATION',
      items: [
        { to: '/authentication/users', label: 'Users', icon: UserCog },
        { to: '/authentication/roles', label: 'Roles', icon: ShieldCheck },
      ],
    })
  }
  // Audit log and feature flags are both system_admin-only server-side
  // (RequireRoles(RoleSystemAdmin), not the dynamic permission system the
  // rest of AUTHENTICATION's group uses) — kept in their own group rather
  // than folded into AUTHENTICATION so its visibility isn't tied to
  // canSeeAuthentication's broader hasPermission check.
  if (isSystemAdmin) {
    groups.push({
      label: 'ADMIN',
      items: [
        { to: '/admin/audit', label: 'Audit log', icon: History },
        { to: '/admin/feature-flags', label: 'Feature flags', icon: Flag },
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
          <TopbarSearch />
          <div className="topbar__spacer" />
          <NotificationBell />
          <UserMenu />
        </header>
        <main className="app-content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
