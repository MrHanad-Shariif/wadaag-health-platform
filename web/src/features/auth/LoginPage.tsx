import { useState, type FormEvent } from 'react'
import { Moon, Sun } from 'lucide-react'
import { ApiError } from '../../api/client'
import { BrandMark } from '../../shared/BrandMark'
import { useTheme } from '../../shared/useTheme'
import { useAuth } from './useAuth'

export function LoginPage() {
  const { login } = useAuth()
  const { theme, toggleTheme } = useTheme()
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await login(identifier, password)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not sign in. Try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-brand-panel">
        <div className="auth-brand-panel__pattern" aria-hidden="true" />
        <div className="auth-brand-panel__content">
          <div className="auth-brand-panel__mark">
            <BrandMark size={36} />
          </div>
          <h1>Wadaag Health</h1>
          <p>Care coordination and telemedicine for physicians, hospitals, labs, pharmacies, and insurers.</p>
        </div>
      </section>

      <section className="auth-form-panel">
        <button
          type="button"
          className="icon-button auth-theme-toggle"
          onClick={toggleTheme}
          aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
        >
          {theme === 'dark' ? <Sun size={18} aria-hidden="true" /> : <Moon size={18} aria-hidden="true" />}
        </button>
        <form className="auth-form" onSubmit={handleSubmit}>
          <h2>Sign in</h2>
          <p className="auth-form__subtitle">Use your facility credentials to continue.</p>

          <label htmlFor="identifier">Email or phone</label>
          <input
            id="identifier"
            type="text"
            autoComplete="username"
            value={identifier}
            onChange={(e) => setIdentifier(e.target.value)}
            required
          />

          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />

          {error && (
            <p role="alert" className="auth-error">
              {error}
            </p>
          )}

          <button type="submit" className="button button--primary" disabled={submitting}>
            {submitting ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
      </section>
    </main>
  )
}
