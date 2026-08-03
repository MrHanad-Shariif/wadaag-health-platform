import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { createConversation } from './api'

export function NewConversationPage() {
  const navigate = useNavigate()
  const { show } = useToast()

  const [otherUserId, setOtherUserId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const conversation = await createConversation(otherUserId.trim())
      show('Conversation started')
      navigate(`/messages/${conversation.id}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not start conversation.')
      setSubmitting(false)
    }
  }

  return (
    <div className="page page--narrow">
      <PageHeader eyebrow="Messages" title="New conversation" />

      <form className="form" onSubmit={handleSubmit}>
        <label htmlFor="otherUserId">Recipient's user ID</label>
        <input
          id="otherUserId"
          value={otherUserId}
          onChange={(e) => setOtherUserId(e.target.value)}
          placeholder="The other person's user ID"
          required
        />
        <p className="form-hint">
          There's no user directory yet — ask the person you want to message for their user ID. If a conversation
          with them already exists, it'll be reused instead of creating a duplicate.
        </p>

        {error && <p role="alert" className="form-error">{error}</p>}

        <button type="submit" disabled={submitting || !otherUserId.trim()}>
          {submitting ? 'Starting…' : 'Start conversation'}
        </button>
      </form>
    </div>
  )
}
