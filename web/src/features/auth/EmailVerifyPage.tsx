import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { CheckCircle2, Moon, Sun } from 'lucide-react'
import { ApiError } from '../../api/client'
import { BrandMark } from '../../shared/BrandMark'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { useTheme } from '../../shared/useTheme'
import { verifyEmail } from './api'

type Status = 'loading' | 'success' | 'error'

export function EmailVerifyPage() {
  const { theme, toggleTheme } = useTheme()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')
  const [status, setStatus] = useState<Status>('loading')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) {
      setStatus('error')
      setError('This link is missing a verification token.')
      return
    }

    let cancelled = false
    setStatus('loading')
    verifyEmail(token)
      .then(() => {
        if (!cancelled) setStatus('success')
      })
      .catch((err) => {
        if (cancelled) return
        setError(err instanceof ApiError ? err.message : 'This link is invalid or has expired.')
        setStatus('error')
      })

    return () => {
      cancelled = true
    }
  }, [token])

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

        <div className="auth-form" style={{ textAlign: 'center', alignItems: 'center' }}>
          <h2>Email verification</h2>

          {status === 'loading' && (
            <div style={{ marginTop: 16 }}>
              <LoadingState />
            </div>
          )}

          {status === 'success' && (
            <>
              <p className="status-message status-message--success" style={{ marginTop: 16, justifyContent: 'center' }}>
                <CheckCircle2 size={16} aria-hidden="true" />
                Your email is verified — you can now sign in.
              </p>
              <Link to="/" className="button button--primary" style={{ marginTop: 22 }}>
                Go to sign in
              </Link>
            </>
          )}

          {status === 'error' && (
            <>
              <div style={{ marginTop: 16 }}>
                <ErrorState message={error ?? 'This link is invalid or has expired.'} />
              </div>
              <Link to="/" className="button button--primary" style={{ marginTop: 22 }}>
                Back to sign in
              </Link>
            </>
          )}
        </div>
      </section>
    </main>
  )
}
