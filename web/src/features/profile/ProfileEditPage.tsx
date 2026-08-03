import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Camera } from 'lucide-react'
import { API_URL, ApiError } from '../../api/client'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { initials } from '../../shared/avatar'
import { useAuth } from '../auth/useAuth'
import { getMyProfile, updateMyProfile, uploadMyProfilePhoto } from './api'
import type { Profile } from './types'

// Fixed set of availability values — keeps the field consistent across
// providers rather than allowing arbitrary free text.
const AVAILABILITY_OPTIONS = ['available', 'busy', 'away', 'offline']

const MAX_PHOTO_BYTES = 5 * 1024 * 1024

function photoSrc(photoUrl: string | null): string | null {
  if (!photoUrl) return null
  return `${API_URL}${photoUrl}`
}

export function ProfileEditPage() {
  const { user } = useAuth()
  const { show } = useToast()
  const state = useFetch(() => getMyProfile(), [])

  const [profile, setProfile] = useState<Profile | null>(null)
  const [bio, setBio] = useState('')
  const [languagesText, setLanguagesText] = useState('')
  const [availabilityStatus, setAvailabilityStatus] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [uploadingPhoto, setUploadingPhoto] = useState(false)
  const [photoError, setPhotoError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!state.data) return
    setProfile(state.data)
    setBio(state.data.bio ?? '')
    setLanguagesText(state.data.languages.join(', '))
    setAvailabilityStatus(state.data.availability_status ?? '')
  }, [state.data])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      const languages = languagesText
        .split(',')
        .map((l) => l.trim())
        .filter(Boolean)
      const updated = await updateMyProfile({
        bio,
        languages,
        availability_status: availabilityStatus,
      })
      setProfile(updated)
      show('Profile updated')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update profile.')
    } finally {
      setSaving(false)
    }
  }

  async function handlePhotoSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setPhotoError(null)

    if (file.size > MAX_PHOTO_BYTES) {
      setPhotoError('Photo must be 5MB or smaller.')
      e.target.value = ''
      return
    }

    setUploadingPhoto(true)
    try {
      const updated = await uploadMyProfilePhoto(file)
      setProfile(updated)
      show('Profile photo updated')
    } catch (err) {
      setPhotoError(err instanceof ApiError ? err.message : 'Could not upload photo.')
    } finally {
      setUploadingPhoto(false)
      e.target.value = ''
    }
  }

  if (state.loading) return <LoadingState />
  if (state.error) return <ErrorState message={state.error} />
  if (!profile) return null

  const name = user?.email ?? user?.phone ?? 'Account'
  const avatarUrl = photoSrc(profile.photo_url)

  return (
    <div className="page page--narrow">
      <PageHeader eyebrow="Account" title="Edit profile" description="Update your bio, languages, availability, and photo." />

      <div className="profile-page__photo-section">
        <div className="profile-page__avatar">
          {avatarUrl ? (
            <img src={avatarUrl} alt="Profile photo" className="profile-page__avatar-img" />
          ) : (
            <span className="profile-page__avatar-fallback">{initials(name)}</span>
          )}
        </div>
        <div className="profile-page__photo-actions">
          <button
            type="button"
            className="profile-page__photo-button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadingPhoto}
          >
            <Camera size={16} aria-hidden="true" />
            {uploadingPhoto ? 'Uploading…' : 'Change photo'}
          </button>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            className="profile-page__file-input"
            onChange={handlePhotoSelect}
          />
          {photoError && (
            <p role="alert" className="form-error">
              {photoError}
            </p>
          )}
        </div>
      </div>

      <form className="form" onSubmit={handleSubmit}>
        <label htmlFor="bio">Bio</label>
        <textarea id="bio" value={bio} onChange={(e) => setBio(e.target.value)} placeholder="Tell colleagues a bit about yourself…" />

        <label htmlFor="languages">Languages</label>
        <input
          id="languages"
          value={languagesText}
          onChange={(e) => setLanguagesText(e.target.value)}
          placeholder="en, so, ar"
        />

        <label htmlFor="availability">Availability</label>
        <select id="availability" value={availabilityStatus} onChange={(e) => setAvailabilityStatus(e.target.value)}>
          <option value="">Not specified</option>
          {AVAILABILITY_OPTIONS.map((opt) => (
            <option key={opt} value={opt}>
              {opt.charAt(0).toUpperCase() + opt.slice(1)}
            </option>
          ))}
        </select>

        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}

        <button type="submit" disabled={saving}>
          {saving ? 'Saving…' : 'Save profile'}
        </button>
      </form>
    </div>
  )
}
