import { Moon, Sun } from 'lucide-react'
import { PageHeader } from '../../shared/PageHeader'
import { useTheme } from '../../shared/useTheme'

// Same useTheme() hook UserMenu's theme toggle already uses — this is an
// additional, discoverable entry point, not a replacement (see
// shared/UserMenu.tsx's own toggle, kept as-is).
export function ThemeSection() {
  const { theme, toggleTheme } = useTheme()

  return (
    <div className="page page--narrow">
      <PageHeader eyebrow="Settings" title="Theme" description="Switch between light and dark mode." />

      <ul className="record-list">
        <li className="feature-flag-row">
          <div>
            <strong>Dark mode</strong>
            <div className="record-list__note">Currently using {theme} theme.</div>
          </div>
          <button type="button" className="button-link" onClick={toggleTheme}>
            {theme === 'dark' ? <Sun size={16} aria-hidden="true" /> : <Moon size={16} aria-hidden="true" />}
            {theme === 'dark' ? 'Switch to light' : 'Switch to dark'}
          </button>
        </li>
      </ul>
    </div>
  )
}
