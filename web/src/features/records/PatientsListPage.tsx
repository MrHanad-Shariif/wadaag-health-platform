import { useNavigate } from 'react-router-dom'
import { useFetch } from '../../shared/useFetch'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { listPatients } from './api'
import type { Patient } from './types'

export function PatientsListPage() {
  const navigate = useNavigate()
  const state = useFetch(() => listPatients(), [])

  const columns: DataTableColumn<Patient>[] = [
    { key: 'full_name', header: 'Name', sortable: true, value: (p) => p.full_name, render: (p) => p.full_name },
    {
      key: 'date_of_birth',
      header: 'Date of birth',
      sortable: true,
      value: (p) => p.date_of_birth ?? '',
      render: (p) => p.date_of_birth ?? <span className="empty-state">—</span>,
    },
    {
      key: 'sex',
      header: 'Sex',
      render: (p) => p.sex ?? <span className="empty-state">—</span>,
    },
    {
      key: 'phone',
      header: 'Phone',
      render: (p) => p.phone ?? <span className="empty-state">—</span>,
    },
    {
      key: 'national_id',
      header: 'National ID',
      render: (p) => p.national_id ?? <span className="empty-state">—</span>,
    },
  ]

  return (
    <div className="page page--wide">
      <PageHeader eyebrow="Records" title="Patients" description="Every patient registered across all facilities." />

      <DataTable
        columns={columns}
        data={state.data}
        loading={state.loading}
        error={state.error}
        getRowKey={(p) => p.id}
        onRowClick={(p) => navigate(`/patients/${p.id}`)}
        searchPlaceholder="Search patients…"
        emptyMessage="No patients registered yet."
      />
    </div>
  )
}
