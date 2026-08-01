export function LoadingState() {
  return <p className="status-message">Loading…</p>
}

export function ErrorState({ message }: { message: string }) {
  return (
    <p className="status-message status-message--error" role="alert">
      {message}
    </p>
  )
}
