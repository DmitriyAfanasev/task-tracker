import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { api } from '../api/client'
import {
  STATUS_LABELS,
  TASK_STATUSES,
  type Role,
  type Task,
  type TaskComment,
  type TaskHistoryEntry,
  type TaskStatus,
  type TeamMemberInfo,
} from '../api/types'
import { useAuth } from '../auth/AuthContext'
import { ErrorAlert } from '../components/ErrorAlert'
import { StatusBadge } from '../components/StatusBadge'
import { formatDate, formatValue } from '../lib/format'

type Tab = 'edit' | 'history' | 'comments'

function canEditField(
  task: Task,
  role: Role,
  userId: number,
  field: 'title' | 'description' | 'status' | 'assignee_id',
): boolean {
  if (role === 'owner' || role === 'admin') return true
  if (task.created_by === userId) return true
  if (task.assignee_id === userId && field === 'status') return true
  return false
}

export function TaskPanel({
  task,
  role,
  teamId,
  onClose,
  onUpdated,
}: {
  task: Task
  role: Role
  teamId: number
  onClose: () => void
  onUpdated: (task: Task) => void
}) {
  const { user } = useAuth()
  const userId = user!.id

  const [tab, setTab] = useState<Tab>('edit')
  const [title, setTitle] = useState(task.title)
  const [description, setDescription] = useState(task.description)
  const [status, setStatus] = useState<TaskStatus>(task.status)
  const [assigneeId, setAssigneeId] = useState(
    task.assignee_id != null ? String(task.assignee_id) : '',
  )
  const [version, setVersion] = useState(task.version)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<unknown>(null)

  const [members, setMembers] = useState<TeamMemberInfo[]>([])
  const [membersLoading, setMembersLoading] = useState(false)

  const assigneeOptions = useMemo(
    () => [
      { value: '', label: 'Не назначен' },
      ...members.map((m) => ({
        value: String(m.user_id),
        label: `${m.name} (#${m.user_id})`,
      })),
    ],
    [members],
  )

  const [history, setHistory] = useState<TaskHistoryEntry[]>([])
  const [historyLoading, setHistoryLoading] = useState(false)

  const [comments, setComments] = useState<TaskComment[]>([])
  const [commentsTotal, setCommentsTotal] = useState(0)
  const [commentsLoading, setCommentsLoading] = useState(false)
  const [commentText, setCommentText] = useState('')
  const [commentPending, setCommentPending] = useState(false)

  useEffect(() => {
    setTitle(task.title)
    setDescription(task.description)
    setStatus(task.status)
    setAssigneeId(task.assignee_id != null ? String(task.assignee_id) : '')
    setVersion(task.version)
    setError(null)
  }, [task])

  useEffect(() => {
    let cancelled = false
    setMembersLoading(true)
    api
      .listTeamMembers(teamId)
      .then((res) => {
        if (!cancelled) setMembers(res.items)
      })
      .catch((err) => {
        if (!cancelled) setError(err)
      })
      .finally(() => {
        if (!cancelled) setMembersLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [teamId])

  useEffect(() => {
    if (tab !== 'history') return
    let cancelled = false
    setHistoryLoading(true)
    api
      .listTaskHistory(task.id)
      .then((res) => {
        if (!cancelled) setHistory(res.items)
      })
      .catch((err) => {
        if (!cancelled) setError(err)
      })
      .finally(() => {
        if (!cancelled) setHistoryLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [tab, task.id])

  useEffect(() => {
    if (tab !== 'comments') return
    let cancelled = false
    setCommentsLoading(true)
    api
      .listTaskComments(task.id)
      .then((res) => {
        if (!cancelled) {
          setComments(res.items)
          setCommentsTotal(res.total)
        }
      })
      .catch((err) => {
        if (!cancelled) setError(err)
      })
      .finally(() => {
        if (!cancelled) setCommentsLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [tab, task.id])

  async function onSave(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    try {
      const body: {
        version: number
        title?: string
        description?: string
        status?: TaskStatus
        assignee_id?: number
      } = { version }

      if (canEditField(task, role, userId, 'title') && title !== task.title) {
        body.title = title.trim()
      }
      if (
        canEditField(task, role, userId, 'description') &&
        description !== task.description
      ) {
        body.description = description
      }
      if (canEditField(task, role, userId, 'status') && status !== task.status) {
        body.status = status
      }
      if (canEditField(task, role, userId, 'assignee_id')) {
        const next =
          assigneeId.trim() === '' ? 0 : Number.parseInt(assigneeId, 10)
        const prev = task.assignee_id ?? 0
        if (!Number.isNaN(next) && next !== prev) {
          body.assignee_id = next
        }
      }

      const updated = await api.updateTask(task.id, body)
      setVersion(updated.version)
      onUpdated(updated)
    } catch (err) {
      setError(err)
    } finally {
      setSaving(false)
    }
  }

  async function onAddComment(e: FormEvent) {
    e.preventDefault()
    if (!commentText.trim()) return
    setCommentPending(true)
    setError(null)
    try {
      const created = await api.addTaskComment(task.id, commentText.trim())
      setComments((prev) => [created, ...prev])
      setCommentsTotal((t) => t + 1)
      setCommentText('')
    } catch (err) {
      setError(err)
    } finally {
      setCommentPending(false)
    }
  }

  return (
    <aside className="task-panel">
      <div className="task-panel-header">
        <div>
          <StatusBadge status={task.status} />
          <h2>Задача #{task.id}</h2>
        </div>
        <button
          type="button"
          className="btn btn-ghost btn-icon"
          onClick={onClose}
          aria-label="Закрыть"
        >
          ×
        </button>
      </div>

      <div className="tabs">
        {(['edit', 'history', 'comments'] as Tab[]).map((t) => (
          <button
            key={t}
            type="button"
            className={`tab ${tab === t ? 'active' : ''}`}
            onClick={() => {
              setError(null)
              setTab(t)
            }}
          >
            {t === 'edit' && 'Редактирование'}
            {t === 'history' && 'История'}
            {t === 'comments' && `Комментарии${commentsTotal ? ` (${commentsTotal})` : ''}`}
          </button>
        ))}
      </div>

      <ErrorAlert error={error} />

      {tab === 'edit' && (
        <form className="form" onSubmit={onSave}>
          <label>
            Заголовок
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={!canEditField(task, role, userId, 'title')}
              required
              maxLength={255}
            />
          </label>
          <label>
            Описание
            <textarea
              rows={5}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={!canEditField(task, role, userId, 'description')}
              maxLength={5000}
            />
          </label>
          <label>
            Статус
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value as TaskStatus)}
              disabled={!canEditField(task, role, userId, 'status')}
            >
              {TASK_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {STATUS_LABELS[s]}
                </option>
              ))}
            </select>
          </label>
          <label>
            Исполнитель
            <select
              value={assigneeId}
              onChange={(e) => setAssigneeId(e.target.value)}
              disabled={!canEditField(task, role, userId, 'assignee_id')}
            >
              {assigneeOptions.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </label>
          {!membersLoading && members.length === 0 && (
            <p className="muted">
              В команде пока нет участников. Пригласите участников на вкладке{' '}
              «Команда».
            </p>
          )}
          <p className="meta">
            Автор: #{task.created_by} · Версия: {version}
            {task.closed_at && ` · Закрыта: ${formatDate(task.closed_at)}`}
          </p>
          <button type="submit" className="btn btn-primary" disabled={saving}>
            {saving ? 'Сохраняем…' : 'Сохранить'}
          </button>
        </form>
      )}

      {tab === 'history' && (
        <div className="stack">
          {historyLoading ? (
            <p className="muted">Загрузка…</p>
          ) : history.length === 0 ? (
            <p className="muted">История пуста.</p>
          ) : (
            history.map((entry) => (
              <article key={entry.id} className="history-entry">
                <header>
                  <strong>#{entry.changed_by}</strong>
                  <time>{formatDate(entry.created_at)}</time>
                </header>
                <ul>
                  {entry.changes.map((c, i) => (
                    <li key={`${entry.id}-${c.field}-${i}`}>
                      <code>{c.field}</code>:{' '}
                      <span className="old">{formatValue(c.old)}</span>
                      {' → '}
                      <span className="new">{formatValue(c.new)}</span>
                    </li>
                  ))}
                </ul>
              </article>
            ))
          )}
        </div>
      )}

      {tab === 'comments' && (
        <div className="stack">
          <form className="form comment-form" onSubmit={onAddComment}>
            <label>
              Новый комментарий
              <textarea
                rows={3}
                required
                minLength={1}
                maxLength={5000}
                value={commentText}
                onChange={(e) => setCommentText(e.target.value)}
                placeholder="Проверил миграции…"
              />
            </label>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={commentPending}
            >
              {commentPending ? 'Отправка…' : 'Добавить'}
            </button>
          </form>
          {commentsLoading ? (
            <p className="muted">Загрузка…</p>
          ) : comments.length === 0 ? (
            <p className="muted">Комментариев пока нет.</p>
          ) : (
            comments.map((c) => (
              <article key={c.id} className="comment">
                <header>
                  <strong>Пользователь #{c.user_id}</strong>
                  <time>{formatDate(c.created_at)}</time>
                </header>
                <p>{c.content}</p>
              </article>
            ))
          )}
        </div>
      )}
    </aside>
  )
}
