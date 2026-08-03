export const WEEKDAY_LABELS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

export function formatStatus(status: string): string {
  return status.replace('_', ' ')
}

export function formatWeekday(weekday: number): string {
  return WEEKDAY_LABELS[weekday] ?? `Day ${weekday}`
}
