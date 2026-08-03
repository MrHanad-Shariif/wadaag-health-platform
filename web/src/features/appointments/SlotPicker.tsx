import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { getSlots } from './api'
import type { Slot } from './types'

interface SlotPickerProps {
  providerId: string
  date: string
  onDateChange: (date: string) => void
  selectedSlot: Slot | null
  onSelectSlot: (slot: Slot | null) => void
  minDate?: string
}

// Shared by the booking flow and the "reschedule" action on the detail
// page: a native date input plus that day's open Slot[] rendered as
// clickable buttons (no calendar-grid component — the roadmap explicitly
// calls this out as unnecessary for the MVP).
export function SlotPicker({ providerId, date, onDateChange, selectedSlot, onSelectSlot, minDate }: SlotPickerProps) {
  const slotsState = useFetch(() => (date ? getSlots(providerId, date) : Promise.resolve([])), [providerId, date])

  return (
    <div className="slot-picker">
      <label htmlFor="appointmentDate">Date</label>
      <input
        id="appointmentDate"
        type="date"
        value={date}
        min={minDate}
        onChange={(e) => {
          onSelectSlot(null)
          onDateChange(e.target.value)
        }}
        required
      />

      {slotsState.loading && <LoadingState />}
      {slotsState.error && <ErrorState message={slotsState.error} />}
      {!slotsState.loading && !slotsState.error && date && (
        (slotsState.data?.length ?? 0) > 0 ? (
          <div className="slot-grid">
            {slotsState.data!.map((slot) => (
              <button
                key={slot.start_at}
                type="button"
                className={`slot-button${selectedSlot?.start_at === slot.start_at ? ' slot-button--selected' : ''}`}
                onClick={() => onSelectSlot(slot)}
              >
                {new Date(slot.start_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
              </button>
            ))}
          </div>
        ) : (
          <p className="empty-state">No open slots on this date.</p>
        )
      )}
    </div>
  )
}
