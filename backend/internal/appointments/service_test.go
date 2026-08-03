package appointments

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// aug3_2026 is a Monday (weekday=1) — used as the fixed reference date
// across every generateSlots test so each test only has to reason about
// times of day, not weekday matching separately.
var aug3_2026 = time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC)

func mustSlots(t *testing.T, start, end string) (time.Time, time.Time) {
	t.Helper()
	s, err := timeOfDayOn(aug3_2026, start)
	if err != nil {
		t.Fatalf("parse start %q: %v", start, err)
	}
	e, err := timeOfDayOn(aug3_2026, end)
	if err != nil {
		t.Fatalf("parse end %q: %v", end, err)
	}
	return s, e
}

func appt(t *testing.T, start, end string) Appointment {
	s, e := mustSlots(t, start, end)
	return Appointment{ID: uuid.New(), StartAt: s, EndAt: e, Status: StatusScheduled}
}

func TestGenerateSlots_NoAvailabilityRules(t *testing.T) {
	slots := generateSlots(nil, nil, aug3_2026, 30)
	if len(slots) != 0 {
		t.Fatalf("expected no slots with no availability rules, got %d", len(slots))
	}
}

func TestGenerateSlots_NoRuleForThatWeekday(t *testing.T) {
	// Tuesday (weekday=2), but we're generating for Monday.
	rules := []AvailabilityRule{{Weekday: 2, StartTime: "09:00", EndTime: "17:00"}}
	slots := generateSlots(rules, nil, aug3_2026, 30)
	if len(slots) != 0 {
		t.Fatalf("expected no slots when no rule matches the requested weekday, got %d", len(slots))
	}
}

func TestGenerateSlots_SimpleWindow(t *testing.T) {
	// 09:00-10:00 in 30-minute slots -> exactly two slots, back to back,
	// with the second one's end landing exactly on the window's end time
	// (the boundary case: a slot exactly filling the remainder must be
	// kept, not dropped for "running past the end").
	rules := []AvailabilityRule{{Weekday: 1, StartTime: "09:00", EndTime: "10:00"}}
	slots := generateSlots(rules, nil, aug3_2026, 30)

	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d: %+v", len(slots), slots)
	}
	wantStart0, wantEnd0 := mustSlots(t, "09:00", "09:30")
	wantStart1, wantEnd1 := mustSlots(t, "09:30", "10:00")
	if !slots[0].StartAt.Equal(wantStart0) || !slots[0].EndAt.Equal(wantEnd0) {
		t.Errorf("slot 0 = %+v, want [%v, %v)", slots[0], wantStart0, wantEnd0)
	}
	if !slots[1].StartAt.Equal(wantStart1) || !slots[1].EndAt.Equal(wantEnd1) {
		t.Errorf("slot 1 = %+v, want [%v, %v)", slots[1], wantStart1, wantEnd1)
	}
}

func TestGenerateSlots_SlotPastWindowEndIsDropped(t *testing.T) {
	// 09:00-09:45 with 30-minute slots: only one slot fits (09:00-09:30);
	// a slot starting at 09:30 would end at 10:00, past the 09:45 window
	// end, and must be dropped entirely rather than truncated to 09:45.
	rules := []AvailabilityRule{{Weekday: 1, StartTime: "09:00", EndTime: "09:45"}}
	slots := generateSlots(rules, nil, aug3_2026, 30)

	if len(slots) != 1 {
		t.Fatalf("expected 1 slot, got %d: %+v", len(slots), slots)
	}
	wantStart, wantEnd := mustSlots(t, "09:00", "09:30")
	if !slots[0].StartAt.Equal(wantStart) || !slots[0].EndAt.Equal(wantEnd) {
		t.Errorf("slot = %+v, want [%v, %v)", slots[0], wantStart, wantEnd)
	}
}

func TestGenerateSlots_ExistingAppointmentSplitsWindow(t *testing.T) {
	// 09:00-11:00 window, a booked 09:30-10:00 appointment splits it —
	// expect 09:00-09:30, then a gap, then 10:00-10:30 and 10:30-11:00.
	rules := []AvailabilityRule{{Weekday: 1, StartTime: "09:00", EndTime: "11:00"}}
	existing := []Appointment{appt(t, "09:30", "10:00")}

	slots := generateSlots(rules, existing, aug3_2026, 30)

	wantRanges := [][2]string{{"09:00", "09:30"}, {"10:00", "10:30"}, {"10:30", "11:00"}}
	if len(slots) != len(wantRanges) {
		t.Fatalf("expected %d slots, got %d: %+v", len(wantRanges), len(slots), slots)
	}
	for i, want := range wantRanges {
		wantStart, wantEnd := mustSlots(t, want[0], want[1])
		if !slots[i].StartAt.Equal(wantStart) || !slots[i].EndAt.Equal(wantEnd) {
			t.Errorf("slot %d = %+v, want [%v, %v)", i, slots[i], wantStart, wantEnd)
		}
	}
}

func TestGenerateSlots_BackToBackAppointmentsLeaveNoGapSlots(t *testing.T) {
	// Two appointments booked back to back (09:00-09:30 and 09:30-10:00)
	// inside a 09:00-10:00 window should consume the whole window, leaving
	// zero available slots — and, symmetrically, an appointment ending
	// exactly when another begins must not be treated as if they overlap
	// each other from the slot-generation side either.
	rules := []AvailabilityRule{{Weekday: 1, StartTime: "09:00", EndTime: "10:00"}}
	existing := []Appointment{
		appt(t, "09:00", "09:30"),
		appt(t, "09:30", "10:00"),
	}

	slots := generateSlots(rules, existing, aug3_2026, 30)
	if len(slots) != 0 {
		t.Fatalf("expected 0 slots, got %d: %+v", len(slots), slots)
	}
}

func TestGenerateSlots_SlotExactlyAtWindowBoundaryIsAvailable(t *testing.T) {
	// A slot whose start lands exactly where a prior appointment ends must
	// be considered free (half-open interval semantics: [9:00,9:30) and
	// [9:30,10:00) don't overlap).
	rules := []AvailabilityRule{{Weekday: 1, StartTime: "09:00", EndTime: "10:00"}}
	existing := []Appointment{appt(t, "09:00", "09:30")}

	slots := generateSlots(rules, existing, aug3_2026, 30)
	if len(slots) != 1 {
		t.Fatalf("expected 1 slot, got %d: %+v", len(slots), slots)
	}
	wantStart, wantEnd := mustSlots(t, "09:30", "10:00")
	if !slots[0].StartAt.Equal(wantStart) || !slots[0].EndAt.Equal(wantEnd) {
		t.Errorf("slot = %+v, want [%v, %v)", slots[0], wantStart, wantEnd)
	}
}

func TestGenerateSlots_CancelledAppointmentsAreNotFilteredHere(t *testing.T) {
	// generateSlots trusts its "existing" input as already being the
	// non-cancelled set (that filtering happens in the
	// ListAppointmentsForProviderInRange SQL query, not here) — so a
	// cancelled appointment passed in would still block a slot. This test
	// just documents that expectation: callers must pre-filter, or, as the
	// real Service.GetAvailableSlots does, only ever pass the query's
	// already-non-cancelled result.
	rules := []AvailabilityRule{{Weekday: 1, StartTime: "09:00", EndTime: "09:30"}}
	start, end := mustSlots(t, "09:00", "09:30")
	existing := []Appointment{{ID: uuid.New(), Status: StatusCancelled, StartAt: start, EndAt: end}}

	slots := generateSlots(rules, existing, aug3_2026, 30)
	if len(slots) != 0 {
		t.Fatalf("expected the passed-in appointment (even if cancelled) to still block the slot, got %d slots", len(slots))
	}
}

func TestGenerateSlots_MultipleRulesSameDay(t *testing.T) {
	// A provider with a split shift: morning and afternoon availability
	// windows on the same weekday both contribute slots.
	rules := []AvailabilityRule{
		{Weekday: 1, StartTime: "09:00", EndTime: "10:00"},
		{Weekday: 1, StartTime: "14:00", EndTime: "15:00"},
	}
	slots := generateSlots(rules, nil, aug3_2026, 30)
	if len(slots) != 4 {
		t.Fatalf("expected 4 slots across both windows, got %d: %+v", len(slots), slots)
	}
}

func TestGenerateSlots_DefaultsSlotMinutesWhenNonPositive(t *testing.T) {
	rules := []AvailabilityRule{{Weekday: 1, StartTime: "09:00", EndTime: "10:00"}}
	slots := generateSlots(rules, nil, aug3_2026, 0)
	if len(slots) != 2 {
		t.Fatalf("expected the 0-minute input to fall back to the 30-minute default (2 slots), got %d", len(slots))
	}
}

func TestGenerateSlots_CustomSlotDuration(t *testing.T) {
	rules := []AvailabilityRule{{Weekday: 1, StartTime: "09:00", EndTime: "10:00"}}
	slots := generateSlots(rules, nil, aug3_2026, 15)
	if len(slots) != 4 {
		t.Fatalf("expected 4 15-minute slots in a 1-hour window, got %d", len(slots))
	}
}

func TestOverlapsAny(t *testing.T) {
	existing := []Appointment{appt(t, "09:00", "09:30")}

	tests := []struct {
		name        string
		start, end  string
		wantOverlap bool
	}{
		{"fully before", "08:00", "08:30", false},
		{"fully after (back to back)", "09:30", "10:00", false},
		{"fully before, ends exactly at existing start", "08:30", "09:00", false},
		{"exact match", "09:00", "09:30", true},
		{"overlaps the start", "08:45", "09:15", true},
		{"overlaps the end", "09:15", "09:45", true},
		{"fully contains existing", "08:45", "09:45", true},
		{"fully contained within existing", "09:10", "09:20", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := mustSlots(t, tt.start, tt.end)
			if got := overlapsAny(start, end, existing); got != tt.wantOverlap {
				t.Errorf("overlapsAny(%s-%s) = %v, want %v", tt.start, tt.end, got, tt.wantOverlap)
			}
		})
	}
}
