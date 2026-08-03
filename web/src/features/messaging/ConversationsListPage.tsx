import { Link, useNavigate } from 'react-router-dom'
import { Plus } from 'lucide-react'
import { useFetch } from '../../shared/useFetch'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { UserDisplayName } from '../../shared/UserDisplayName'
import { useAuth } from '../auth/useAuth'
import { listConversations } from './api'
import { formatRelativeTime } from './format'
import { isUnread, otherPartyId, otherPartyLabel, type Conversation } from './types'

export function ConversationsListPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const state = useFetch(() => listConversations(), [])

  const columns: DataTableColumn<Conversation>[] = [
    {
      key: 'unread',
      header: '',
      value: (c) => (isUnread(c) ? 1 : 0),
      render: (c) => (isUnread(c) ? <span className="badge badge--accent">New</span> : null),
    },
    {
      key: 'other_party',
      header: 'With',
      sortable: true,
      value: (c) => otherPartyLabel(c, user?.id),
      render: (c) => <UserDisplayName userId={otherPartyId(c, user?.id)} />,
    },
    {
      key: 'preview',
      header: 'Last message',
      value: (c) => c.last_message_body ?? '',
      render: (c) => c.last_message_body ?? <span className="empty-state">No messages yet.</span>,
    },
    {
      key: 'last_message_at',
      header: 'Updated',
      sortable: true,
      value: (c) => c.last_message_at ?? c.created_at,
      render: (c) => formatRelativeTime(c.last_message_at ?? c.created_at),
    },
  ]

  return (
    <div className="page page--wide">
      <PageHeader
        eyebrow="Messages"
        title="Conversations"
        description="Direct messages with other users on the platform."
        actions={
          <Link className="button-link" to="/messages/new">
            <Plus size={16} aria-hidden="true" />
            New conversation
          </Link>
        }
      />

      <DataTable
        columns={columns}
        data={state.data}
        loading={state.loading}
        error={state.error}
        getRowKey={(c) => c.id}
        onRowClick={(c) => navigate(`/messages/${c.id}`)}
        searchPlaceholder="Search conversations…"
        emptyMessage="No conversations yet."
      />
    </div>
  )
}
