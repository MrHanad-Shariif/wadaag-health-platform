import { AlertCircle, Loader2 } from 'lucide-react'

export function LoadingState() {
  return (
    <p className="status-message">
      <Loader2 size={16} className="spinner" aria-hidden="true" />
      Loading…
    </p>
  )
}

export function ErrorState({ message }: { message: string }) {
  return (
    <p className="status-message status-message--error" role="alert">
      <AlertCircle size={16} aria-hidden="true" />
      {message}
    </p>
  )
}
