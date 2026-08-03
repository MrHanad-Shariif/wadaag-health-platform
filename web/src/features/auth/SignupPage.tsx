import { useState, type FormEvent } from 'react'
import { Moon, Sun } from 'lucide-react'
import { ApiError } from '../../api/client'
import { BrandMark } from '../../shared/BrandMark'
import { useTheme } from '../../shared/useTheme'
import { useAuth } from './useAuth'
import { register } from './api'

export function SignupPage({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const { login } = useAuth()
  const { theme, toggleTheme } = useTheme()
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)

    if (password !== confirmPassword) {
      setError('Passwords do not match.')
      return
    }

    setSubmitting(true)
    try {
      await register(email, password, fullName)
      await login(email, password)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not create account. Try again.')
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
          <h2>Create your account</h2>
          <p className="auth-form__subtitle">Sign up as a patient and choose your own password.</p>

          <label htmlFor="fullName">Full name</label>
          <input
            id="fullName"
            type="text"
            autoComplete="name"
            value={fullName}
            onChange={(e) => setFullName(e.target.value)}
            required
          />

          <label htmlFor="email">Email</label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />

          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            minLength={8}
            required
          />

          <label htmlFor="confirmPassword">Confirm password</label>
          <input
            id="confirmPassword"
            type="password"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            minLength={8}
            required
          />

          {error && (
            <p role="alert" className="auth-error">
              {error}
            </p>
          )}

          <button type="submit" className="button button--primary" disabled={submitting}>
            {submitting ? 'Creating account…' : 'Create account'}
          </button>

          <button type="button" className="button button--link" onClick={onSwitchToLogin}>
            Already have an account? Sign in
          </button>
        </form>
      </section>
    </main>
  )
}
