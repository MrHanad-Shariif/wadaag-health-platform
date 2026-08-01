import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { createPatient } from './api'

export function NewPatientPage() {
  const navigate = useNavigate()
  const [fullName, setFullName] = useState('')
  const [dateOfBirth, setDateOfBirth] = useState('')
  const [sex, setSex] = useState('')
  const [phone, setPhone] = useState('')
  const [nationalId, setNationalId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const patient = await createPatient({
        full_name: fullName,
        date_of_birth: dateOfBirth || undefined,
        sex: sex || undefined,
        phone: phone || undefined,
        national_id: nationalId || undefined,
      })
      navigate(`/patients/${patient.id}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not register patient.')
      setSubmitting(false)
    }
  }

  return (
    <div className="page page--narrow">
      <p className="page-eyebrow">Records</p>
      <h1>Register a patient</h1>
      <form className="form" onSubmit={handleSubmit}>
        <label htmlFor="fullName">Full name</label>
        <input id="fullName" value={fullName} onChange={(e) => setFullName(e.target.value)} required />

        <label htmlFor="dob">Date of birth</label>
        <input id="dob" type="date" value={dateOfBirth} onChange={(e) => setDateOfBirth(e.target.value)} />

        <label htmlFor="sex">Sex</label>
        <select id="sex" value={sex} onChange={(e) => setSex(e.target.value)}>
          <option value="">Not specified</option>
          <option value="female">Female</option>
          <option value="male">Male</option>
        </select>

        <label htmlFor="phone">Phone</label>
        <input id="phone" value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+252 6XX XXX XXX" />

        <label htmlFor="nationalId">National ID</label>
        <input id="nationalId" value={nationalId} onChange={(e) => setNationalId(e.target.value)} />

        {error && <p role="alert" className="form-error">{error}</p>}

        <button type="submit" disabled={submitting}>
          {submitting ? 'Saving…' : 'Register patient'}
        </button>
      </form>
    </div>
  )
}
