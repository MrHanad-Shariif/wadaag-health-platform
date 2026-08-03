import { useState, type FormEvent } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { UserDisplayName } from '../../shared/UserDisplayName'
import { ProviderDisplayName } from '../providers/ProviderDisplayName'
import { useAuth } from '../auth/useAuth'
import { formatStatus } from './format'
import { closeConsult, getConsult, listMessages, postMessage } from './api'

export function ConsultDetailPage() {
  const { consultationId } = useParams<{ consultationId: string }>()
  const { user } = useAuth()
  const { show } = useToast()

  const consultState = useFetch(() => getConsult(consultationId!), [consultationId])
  const messagesState = useFetch(() => listMessages(consultationId!), [consultationId])

  const [body, setBody] = useState('')
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState<string | null>(null)
  const [closing, setClosing] = useState(false)
  const [closeError, setCloseError] = useState<string | null>(null)

  async function handleSend(e: FormEvent) {
    e.preventDefault()
    if (!body.trim()) return
    setSendError(null)
    setSending(true)
    try {
      await postMessage(consultationId!, { body })
      setBody('')
      messagesState.reload()
      consultState.reload()
    } catch (err) {
      setSendError(err instanceof ApiError ? err.message : 'Could not send message.')
    } finally {
      setSending(false)
    }
  }

  async function handleClose() {
    setCloseError(null)
    setClosing(true)
    try {
      await closeConsult(consultationId!)
      consultState.reload()
      show('Consultation closed')
    } catch (err) {
      setCloseError(err instanceof ApiError ? err.message : 'Could not close consultation.')
    } finally {
      setClosing(false)
    }
  }

  if (consultState.loading) return <LoadingState />
  if (consultState.error) return <ErrorState message={consultState.error} />
  const consult = consultState.data!

  const isParty = user?.id === consult.requesting_provider_id || user?.id === consult.target_provider_id
  const canClose = isParty && consult.status !== 'closed'

  return (
    <div className="page">
      <PageHeader
        breadcrumb={[{ label: 'Patient', to: `/patients/${consult.patient_id}` }, { label: 'Consultation' }]}
        title="Consultation"
        description={consult.reason}
      />
      <p>
        <span className={`status-pill status-pill--${consult.status}`}>{formatStatus(consult.status)}</span>
      </p>
      <p>
        Requested by <ProviderDisplayName providerId={consult.requesting_provider_id} /> for{' '}
        <ProviderDisplayName providerId={consult.target_provider_id} />
      </p>

      {canClose && (
        <div className="action-row">
          <button type="button" className="button button--danger" disabled={closing} onClick={handleClose}>
            {closing ? 'Closing…' : 'Close consultation'}
          </button>
        </div>
      )}
      {closeError && <ErrorState message={closeError} />}

      <section>
        <h2>Messages</h2>
        {messagesState.loading && <LoadingState />}
        {messagesState.error && <ErrorState message={messagesState.error} />}
        {messagesState.data && (
          <div className="consult-thread">
            {messagesState.data.length === 0 && <p className="empty-state">No messages yet.</p>}
            {messagesState.data.map((m) => (
              <div
                key={m.id}
                className={`consult-thread__message${m.sender_id === user?.id ? ' consult-thread__message--mine' : ''}`}
              >
                <div className="consult-thread__meta">
                  <span>{m.sender_id === user?.id ? 'You' : <UserDisplayName userId={m.sender_id} />}</span>
                  <span>{new Date(m.created_at).toLocaleString()}</span>
                </div>
                <div className="consult-thread__body">{m.body}</div>
              </div>
            ))}
          </div>
        )}

        {consult.status !== 'closed' && isParty && (
          <form className="form" onSubmit={handleSend}>
            <label htmlFor="messageBody">Reply</label>
            <textarea
              id="messageBody"
              value={body}
              onChange={(e) => setBody(e.target.value)}
              rows={3}
              required
            />
            {sendError && <p role="alert" className="form-error">{sendError}</p>}
            <button type="submit" disabled={sending}>
              {sending ? 'Sending…' : 'Send'}
            </button>
          </form>
        )}
      </section>
    </div>
  )
}
