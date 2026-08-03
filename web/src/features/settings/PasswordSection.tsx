import { useState, type FormEvent } from 'react'
import { ApiError } from '../../api/client'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { changePassword } from '../auth/api'

// Same minimum-length rule the signup form enforces client-side
// (backend/internal/identity/service.go's minPasswordLength) — kept in
// sync manually since there's no shared constants module between the two.
const MIN_PASSWORD_LENGTH = 8

export function PasswordSection() {
  const { show } = useToast()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)

    if (newPassword !== confirmPassword) {
      setError('New passwords do not match.')
      return
    }

    setSaving(true)
    try {
      await changePassword(currentPassword, newPassword)
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      show('Password updated. Your other sessions have been signed out.')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not change password.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="page page--narrow">
      <PageHeader
        eyebrow="Settings"
        title="Password"
        description="Changing your password signs every other active session out."
      />

      <form className="form" onSubmit={handleSubmit}>
        <label htmlFor="currentPassword">Current password</label>
        <input
          id="currentPassword"
          type="password"
          autoComplete="current-password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          required
        />

        <label htmlFor="newPassword">New password</label>
        <input
          id="newPassword"
          type="password"
          autoComplete="new-password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          minLength={MIN_PASSWORD_LENGTH}
          required
        />
        <p className="form-hint">At least {MIN_PASSWORD_LENGTH} characters.</p>

        <label htmlFor="confirmNewPassword">Confirm new password</label>
        <input
          id="confirmNewPassword"
          type="password"
          autoComplete="new-password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          minLength={MIN_PASSWORD_LENGTH}
          required
        />

        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}

        <button type="submit" disabled={saving}>
          {saving ? 'Updating…' : 'Update password'}
        </button>
      </form>
    </div>
  )
}
