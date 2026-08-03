import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '../../shared/PageHeader'
import { ProfileEditPage } from '../profile/ProfileEditPage'
import { SessionsPage } from '../auth/SessionsPage'
import { PasswordSection } from './PasswordSection'
import { NotificationsSection } from './NotificationsSection'
import { ThemeSection } from './ThemeSection'

const TABS = [
  { key: 'profile', label: 'Profile' },
  { key: 'password', label: 'Password' },
  { key: 'notifications', label: 'Notifications' },
  { key: 'theme', label: 'Theme' },
  { key: 'sessions', label: 'Sessions' },
] as const

type TabKey = (typeof TABS)[number]['key']

// Consolidates account-level pages that used to be scattered (Profile at
// "/profile", Sessions at "/settings/sessions") plus two brand-new sections
// (Password, Notifications) into one tabbed area. Each tab renders its own
// full sub-page below the shared tab bar rather than a shared wrapper
// swallowing them, since Profile/Sessions are reused components that each
// already render their own PageHeader.
export function SettingsPage() {
  const { tab } = useParams<{ tab?: string }>()
  const navigate = useNavigate()
  const activeTab: TabKey = TABS.some((t) => t.key === tab) ? (tab as TabKey) : 'profile'

  return (
    <>
      <div className="page page--wide settings-shell">
        <PageHeader
          eyebrow="Account"
          title="Settings"
          description="Manage your profile, security, notifications, and appearance."
        />

        <div className="settings-tabs" role="tablist">
          {TABS.map((t) => (
            <button
              key={t.key}
              type="button"
              role="tab"
              aria-selected={activeTab === t.key}
              className={`settings-tab${activeTab === t.key ? ' settings-tab--active' : ''}`}
              onClick={() => navigate(`/settings/${t.key}`)}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {activeTab === 'profile' && <ProfileEditPage />}
      {activeTab === 'password' && <PasswordSection />}
      {activeTab === 'notifications' && <NotificationsSection />}
      {activeTab === 'theme' && <ThemeSection />}
      {activeTab === 'sessions' && <SessionsPage />}
    </>
  )
}
