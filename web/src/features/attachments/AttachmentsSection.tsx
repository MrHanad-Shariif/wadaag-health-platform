import { useEffect, useRef, useState, type ChangeEvent, type FormEvent } from 'react'
import { Download, FileText, History, Image as ImageIcon, Upload } from 'lucide-react'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { useToast } from '../../shared/useToast'
import { formatBytes } from '../../shared/format'
import { getAttachmentBlob, listAttachments, listVersions, uploadAttachment } from './api'
import type { Attachment, AttachmentMimeType } from './types'

const ALLOWED_MIME_TYPES: AttachmentMimeType[] = ['image/jpeg', 'image/png', 'image/webp', 'application/pdf']
const ACCEPT_ATTR = ALLOWED_MIME_TYPES.join(',')

function isAllowedType(type: string): type is AttachmentMimeType {
  return (ALLOWED_MIME_TYPES as string[]).includes(type)
}

async function downloadAttachment(attachmentId: string, filename: string) {
  const blob = await getAttachmentBlob(attachmentId)
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

export function AttachmentsSection({ patientId, isProvider }: { patientId: string; isProvider: boolean }) {
  const { show } = useToast()
  const attachmentsState = useFetch(() => listAttachments(patientId), [patientId])

  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)

  async function handleUpload(e: FormEvent) {
    e.preventDefault()
    const file = fileInputRef.current?.files?.[0]
    if (!file) {
      setUploadError('Choose a file to upload.')
      return
    }
    if (!isAllowedType(file.type)) {
      setUploadError('Unsupported file type. Allowed: JPEG, PNG, WEBP, PDF.')
      return
    }

    setUploadError(null)
    setUploading(true)
    try {
      await uploadAttachment(patientId, file)
      if (fileInputRef.current) fileInputRef.current.value = ''
      attachmentsState.reload()
      show('Attachment uploaded')
    } catch (err) {
      setUploadError(err instanceof ApiError ? err.message : 'Could not upload attachment.')
    } finally {
      setUploading(false)
    }
  }

  async function handleUploadNewVersion(attachmentId: string, file: File) {
    if (!isAllowedType(file.type)) {
      setUploadError('Unsupported file type. Allowed: JPEG, PNG, WEBP, PDF.')
      return
    }
    setUploadError(null)
    setUploading(true)
    try {
      await uploadAttachment(patientId, file, { supersedesId: attachmentId })
      attachmentsState.reload()
      show('New version uploaded')
    } catch (err) {
      setUploadError(err instanceof ApiError ? err.message : 'Could not upload new version.')
    } finally {
      setUploading(false)
    }
  }

  return (
    <section>
      <h2>Attachments</h2>
      {attachmentsState.loading && <LoadingState />}
      {attachmentsState.error && <ErrorState message={attachmentsState.error} />}

      {attachmentsState.data && (
        attachmentsState.data.length ? (
          <ul className="record-list">
            {attachmentsState.data.map((attachment) => (
              <AttachmentRow
                key={attachment.id}
                attachment={attachment}
                isProvider={isProvider}
                uploading={uploading}
                onUploadNewVersion={handleUploadNewVersion}
              />
            ))}
          </ul>
        ) : (
          <p className="empty-state">No attachments yet.</p>
        )
      )}

      {isProvider && (
        <form className="form form--inline" onSubmit={handleUpload}>
          <label htmlFor="attachmentFile">Upload attachment</label>
          <input id="attachmentFile" ref={fileInputRef} type="file" accept={ACCEPT_ATTR} />
          <button type="submit" disabled={uploading}>
            {uploading ? 'Uploading…' : 'Upload'}
          </button>
          {uploadError && <p role="alert" className="form-error">{uploadError}</p>}
        </form>
      )}
    </section>
  )
}

function AttachmentRow({
  attachment,
  isProvider,
  uploading,
  onUploadNewVersion,
}: {
  attachment: Attachment
  isProvider: boolean
  uploading: boolean
  onUploadNewVersion: (attachmentId: string, file: File) => void
}) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)
  const [showVersions, setShowVersions] = useState(false)
  const versionInputRef = useRef<HTMLInputElement>(null)

  const versionsState = useFetch(
    () => (showVersions ? listVersions(attachment.id) : Promise.resolve<Attachment[] | null>(null)),
    [attachment.id, showVersions],
  )

  useEffect(() => {
    return () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl)
    }
  }, [previewUrl])

  const isImage = attachment.mime_type.startsWith('image/')
  const isPdf = attachment.mime_type === 'application/pdf'
  const previewable = isImage || isPdf

  async function togglePreview() {
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl)
      setPreviewUrl(null)
      return
    }
    setActionError(null)
    setPreviewLoading(true)
    try {
      const blob = await getAttachmentBlob(attachment.id)
      setPreviewUrl(URL.createObjectURL(blob))
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'Could not load preview.')
    } finally {
      setPreviewLoading(false)
    }
  }

  async function handleDownload() {
    setActionError(null)
    try {
      await downloadAttachment(attachment.id, attachment.filename)
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'Could not download attachment.')
    }
  }

  function handleVersionFileChange(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) onUploadNewVersion(attachment.id, file)
    e.target.value = ''
  }

  return (
    <li>
      <div className="attachment-row__header">
        {isImage ? <ImageIcon size={16} aria-hidden="true" /> : <FileText size={16} aria-hidden="true" />}
        <span className="attachment-row__filename">{attachment.filename}</span>
        <span className="record-list__note">
          v{attachment.version} · {formatBytes(attachment.size)} · {new Date(attachment.created_at).toLocaleString()}
        </span>
      </div>

      <div className="form--inline">
        {previewable && (
          <button type="button" className="button-link" onClick={togglePreview} disabled={previewLoading}>
            {previewLoading ? 'Loading…' : previewUrl ? 'Hide preview' : 'Preview'}
          </button>
        )}
        <button type="button" className="button-link" onClick={handleDownload}>
          <Download size={14} aria-hidden="true" /> Download
        </button>
        <button type="button" className="button-link" onClick={() => setShowVersions((v) => !v)}>
          <History size={14} aria-hidden="true" /> {showVersions ? 'Hide versions' : 'Version history'}
        </button>
        {isProvider && (
          <>
            <input
              ref={versionInputRef}
              type="file"
              accept={ACCEPT_ATTR}
              style={{ display: 'none' }}
              onChange={handleVersionFileChange}
            />
            <button
              type="button"
              className="button-link"
              onClick={() => versionInputRef.current?.click()}
              disabled={uploading}
            >
              <Upload size={14} aria-hidden="true" /> Upload new version
            </button>
          </>
        )}
      </div>

      {actionError && <p role="alert" className="form-error">{actionError}</p>}

      {previewUrl && isImage && (
        <img src={previewUrl} alt={attachment.filename} className="attachment-preview__image" />
      )}
      {previewUrl && isPdf && (
        <iframe src={previewUrl} title={attachment.filename} className="attachment-preview__pdf" />
      )}

      {showVersions && (
        <div className="attachment-versions">
          {versionsState.loading && <LoadingState />}
          {versionsState.error && <ErrorState message={versionsState.error} />}
          {versionsState.data && (
            <ul className="record-list">
              {versionsState.data.map((version) => (
                <VersionRow key={version.id} version={version} />
              ))}
            </ul>
          )}
        </div>
      )}
    </li>
  )
}

function VersionRow({ version }: { version: Attachment }) {
  const [error, setError] = useState<string | null>(null)

  async function handleDownload() {
    setError(null)
    try {
      await downloadAttachment(version.id, version.filename)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not download version.')
    }
  }

  return (
    <li>
      <div className="attachment-row__header">
        <span className="attachment-row__filename">v{version.version} — {version.filename}</span>
        <span className="record-list__note">
          {formatBytes(version.size)} · {new Date(version.created_at).toLocaleString()}
        </span>
      </div>
      <button type="button" className="button-link" onClick={handleDownload}>
        <Download size={14} aria-hidden="true" /> Download
      </button>
      {error && <p role="alert" className="form-error">{error}</p>}
    </li>
  )
}
