import {
  useMemo,
  useState,
  type DragEvent,
  type FormEvent,
} from 'react'
import { api } from '../api/client'
import {
  STATUS_LABELS,
  TASK_STATUSES,
  type Task,
  type TaskStatus,
} from '../api/types'
import { ErrorAlert } from '../components/ErrorAlert'
import { formatDate } from '../lib/format'

interface KanbanBoardProps {
  tasks: Task[]
  selectedId: number | null
  busy: boolean
  onSelect: (task: Task) => void
  onTasksChange: (updater: (prev: Task[]) => Task[]) => void
  onCreated: (task: Task) => void
  onError: (err: unknown) => void
  teamId: number
}

export function KanbanBoard({
  tasks,
  selectedId,
  busy,
  onSelect,
  onTasksChange,
  onCreated,
  onError,
  teamId,
}: KanbanBoardProps) {
  const [draggingId, setDraggingId] = useState<number | null>(null)
  const [overStatus, setOverStatus] = useState<TaskStatus | null>(null)
  const [movingId, setMovingId] = useState<number | null>(null)
  const [quickAdd, setQuickAdd] = useState<TaskStatus | null>(null)
  const [quickTitle, setQuickTitle] = useState('')
  const [quickPending, setQuickPending] = useState(false)
  const [moveError, setMoveError] = useState<unknown>(null)

  const byStatus = useMemo(() => {
    const map = Object.fromEntries(
      TASK_STATUSES.map((s) => [s, [] as Task[]]),
    ) as Record<TaskStatus, Task[]>
    for (const task of tasks) {
      map[task.status].push(task)
    }
    for (const status of TASK_STATUSES) {
      map[status].sort(
        (a, b) =>
          new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
      )
    }
    return map
  }, [tasks])

  function onDragStart(e: DragEvent, task: Task) {
    e.dataTransfer.setData('text/task-id', String(task.id))
    e.dataTransfer.effectAllowed = 'move'
    setDraggingId(task.id)
    setMoveError(null)
  }

  function onDragEnd() {
    setDraggingId(null)
    setOverStatus(null)
  }

  function onDragOverColumn(e: DragEvent, status: TaskStatus) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    if (overStatus !== status) setOverStatus(status)
  }

  async function onDropColumn(e: DragEvent, status: TaskStatus) {
    e.preventDefault()
    setOverStatus(null)
    const raw = e.dataTransfer.getData('text/task-id')
    const id = Number.parseInt(raw, 10)
    setDraggingId(null)
    if (!Number.isFinite(id)) return

    const task = tasks.find((t) => t.id === id)
    if (!task || task.status === status) return

    const previous = task
    const optimistic: Task = {
      ...task,
      status,
      updated_at: new Date().toISOString(),
    }
    onTasksChange((prev) => prev.map((t) => (t.id === id ? optimistic : t)))
    setMovingId(id)
    setMoveError(null)

    try {
      const updated = await api.updateTask(id, {
        version: task.version,
        status,
      })
      onTasksChange((prev) => prev.map((t) => (t.id === id ? updated : t)))
    } catch (err) {
      onTasksChange((prev) => prev.map((t) => (t.id === id ? previous : t)))
      setMoveError(err)
      onError(err)
    } finally {
      setMovingId(null)
    }
  }

  async function onQuickCreate(e: FormEvent, status: TaskStatus) {
    e.preventDefault()
    if (!quickTitle.trim()) return
    setQuickPending(true)
    setMoveError(null)
    try {
      const created = await api.createTask({
        team_id: teamId,
        title: quickTitle.trim(),
        status,
      })
      onTasksChange((prev) => [created, ...prev])
      onCreated(created)
      setQuickTitle('')
      setQuickAdd(null)
    } catch (err) {
      setMoveError(err)
      onError(err)
    } finally {
      setQuickPending(false)
    }
  }

  return (
    <div className="kanban">
      <ErrorAlert error={moveError} />
      <div className="kanban-board" aria-busy={busy}>
        {TASK_STATUSES.map((status) => {
          const columnTasks = byStatus[status]
          return (
            <section
              key={status}
              className={`kanban-column kanban-column-${status} ${
                overStatus === status ? 'is-over' : ''
              }`}
              onDragOver={(e) => onDragOverColumn(e, status)}
              onDragLeave={() => {
                if (overStatus === status) setOverStatus(null)
              }}
              onDrop={(e) => void onDropColumn(e, status)}
            >
              <header className="kanban-column-header">
                <div>
                  <h2>{STATUS_LABELS[status]}</h2>
                  <span className="kanban-count">{columnTasks.length}</span>
                </div>
                <button
                  type="button"
                  className="btn btn-ghost btn-icon"
                  title="Добавить задачу"
                  aria-label={`Добавить в «${STATUS_LABELS[status]}»`}
                  onClick={() => {
                    setQuickAdd(status)
                    setQuickTitle('')
                  }}
                >
                  +
                </button>
              </header>

              {quickAdd === status && (
                <form
                  className="kanban-quick-add"
                  onSubmit={(e) => void onQuickCreate(e, status)}
                >
                  <input
                    autoFocus
                    required
                    maxLength={255}
                    placeholder="Название задачи"
                    value={quickTitle}
                    onChange={(e) => setQuickTitle(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Escape') setQuickAdd(null)
                    }}
                  />
                  <div className="form-actions">
                    <button
                      type="button"
                      className="btn btn-ghost"
                      onClick={() => setQuickAdd(null)}
                    >
                      Отмена
                    </button>
                    <button
                      type="submit"
                      className="btn btn-primary"
                      disabled={quickPending}
                    >
                      Добавить
                    </button>
                  </div>
                </form>
              )}

              <ul className="kanban-cards">
                {columnTasks.length === 0 && !quickAdd ? (
                  <li className="kanban-empty">Перетащите карточку сюда</li>
                ) : (
                  columnTasks.map((task) => (
                    <li key={task.id}>
                      <article
                        className={`kanban-card ${
                          selectedId === task.id ? 'active' : ''
                        } ${draggingId === task.id ? 'dragging' : ''} ${
                          movingId === task.id ? 'moving' : ''
                        }`}
                        draggable={movingId == null}
                        onDragStart={(e) => onDragStart(e, task)}
                        onDragEnd={onDragEnd}
                        onClick={() => onSelect(task)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault()
                            onSelect(task)
                          }
                        }}
                        role="button"
                        tabIndex={0}
                      >
                        <h3>{task.title}</h3>
                        {task.description ? (
                          <p className="kanban-card-desc">{task.description}</p>
                        ) : null}
                        <footer className="kanban-card-meta">
                          <span>
                            {task.assignee_id != null
                              ? `#${task.assignee_id}`
                              : 'Без исполнителя'}
                          </span>
                          <time dateTime={task.updated_at}>
                            {formatDate(task.updated_at)}
                          </time>
                        </footer>
                      </article>
                    </li>
                  ))
                )}
              </ul>
            </section>
          )
        })}
      </div>
    </div>
  )
}
