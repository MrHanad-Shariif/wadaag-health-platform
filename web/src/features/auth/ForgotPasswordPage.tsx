import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { CheckCircle2, Moon, Sun } from 'lucide-react'
import { BrandMark } from '../../shared/BrandMark'
import { useTheme } from '../../shared/useTheme'
import { forgotPassword } from './api'

export function ForgotPasswordPage() {
  const { theme, toggleTheme } = useTheme()
  const [identifier, setIdentifier] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [submitted, setSubmitted] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    try {
      await forgotPassword(identifier)
    } catch {
      // Intentionally ignored: the backend always responds 204 regardless of
      // whether the identifier matches an account, and we don't want the UI
      // to distinguish "found" vs "not found" via a different error state.
    } finally {
      setSubmitting(false)
      setSubmitted(true)
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

        {submitted ? (
          <div className="auth-form" style={{ textAlign: 'center', alignItems: 'center' }}>
            <h2>Check your account</h2>
            <p
              className="status-message status-message--success"
              style={{ marginTop: 16, justifyContent: 'center' }}
            >
              <CheckCircle2 size={16} aria-hidden="true" />
              If an account exists for that email or phone, a password reset link has been sent.
            </p>
            <Link to="/" className="button button--primary" style={{ marginTop: 22 }}>
              Back to sign in
            </Link>
          </div>
        ) : (
          <form className="auth-form" onSubmit={handleSubmit}>
            <h2>Forgot password</h2>
            <p className="auth-form__subtitle">
              Enter your account email or phone and we'll send you a link to reset your password.
            </p>

            <label htmlFor="identifier">Email or phone</label>
            <input
              id="identifier"
              type="text"
              autoComplete="username"
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              required
            />

            <button type="submit" className="button button--primary" disabled={submitting}>
              {submitting ? 'Sending…' : 'Send reset link'}
            </button>

            <Link to="/" className="button button--link">
              Back to sign in
            </Link>
          </form>
        )}
      </section>
    </main>
  )
}
