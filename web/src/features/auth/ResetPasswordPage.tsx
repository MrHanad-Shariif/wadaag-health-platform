import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { CheckCircle2, Moon, Sun } from 'lucide-react'
import { ApiError } from '../../api/client'
import { BrandMark } from '../../shared/BrandMark'
import { ErrorState } from '../../shared/StatusMessage'
import { useTheme } from '../../shared/useTheme'
import { resetPassword } from './api'

export function ResetPasswordPage() {
  const { theme, toggleTheme } = useTheme()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [success, setSuccess] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)

    if (newPassword.length < 8) {
      setError('Password must be at least 8 characters.')
      return
    }

    if (newPassword !== confirmPassword) {
      setError('Passwords do not match.')
      return
    }

    setSubmitting(true)
    try {
      await resetPassword(token!, newPassword)
      setSuccess(true)
    } catch (err) {
      setError(
        err instanceof ApiError
          ? 'This reset link is invalid or has expired. Please request a new one.'
          : 'Could not reset password. Try again.',
      )
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

        {!token ? (
          <div className="auth-form" style={{ textAlign: 'center', alignItems: 'center' }}>
            <h2>Reset password</h2>
            <div style={{ marginTop: 16 }}>
              <ErrorState message="This link is missing a reset token. Please request a new password reset link." />
            </div>
            <Link to="/forgot-password" className="button button--primary" style={{ marginTop: 22 }}>
              Request a new link
            </Link>
          </div>
        ) : success ? (
          <div className="auth-form" style={{ textAlign: 'center', alignItems: 'center' }}>
            <h2>Password reset</h2>
            <p
              className="status-message status-message--success"
              style={{ marginTop: 16, justifyContent: 'center' }}
            >
              <CheckCircle2 size={16} aria-hidden="true" />
              Your password has been reset. You can now sign in with your new password.
            </p>
            <Link to="/" className="button button--primary" style={{ marginTop: 22 }}>
              Go to sign in
            </Link>
          </div>
        ) : (
          <form className="auth-form" onSubmit={handleSubmit}>
            <h2>Reset password</h2>
            <p className="auth-form__subtitle">Choose a new password for your account.</p>

            <label htmlFor="newPassword">New password</label>
            <input
              id="newPassword"
              type="password"
              autoComplete="new-password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              minLength={8}
              required
            />

            <label htmlFor="confirmPassword">Confirm new password</label>
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
              {submitting ? 'Resetting…' : 'Reset password'}
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
