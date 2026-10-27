import { STATUS_LABELS, type TaskStatus } from '../api/types'

export function StatusBadge({ status }: { status: TaskStatus }) {
  return (
    <span className={`status-badge status-${status}`}>
      {STATUS_LABELS[status]}
    </span>
  )
}
