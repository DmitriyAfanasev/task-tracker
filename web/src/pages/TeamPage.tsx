import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import {
  ROLE_LABELS,
  STATUS_LABELS,
  TASK_STATUSES,
  type InviteRole,
  type Role,
  type Task,
  type TasksResponse,
  type TaskStatus,
  type TeamMemberInfo,
  type TeamStatsResponse,
  type TeamWithRole,
} from '../api/types'
import { ErrorAlert } from '../components/ErrorAlert'
import { Modal } from '../components/Modal'
import { formatDuration } from '../lib/format'
import { KanbanBoard } from './KanbanBoard'
import { TaskPanel } from './TaskPanel'

type View = 'board' | 'stats'

const FETCH_LIMIT = 100

async function fetchAllTasks(
  teamId: number,
  assigneeId: number | undefined,
  signal?: AbortSignal,
): Promise<{ items: Task[]; total: number }> {
  // Сначала получаем первую страницу, чтобы узнать total.
  const first = await api.listTasks({
    team_id: teamId,
    assignee_id: assigneeId,
    limit: FETCH_LIMIT,
    offset: 0,
  })
  if (signal?.aborted) return { items: [], total: 0 }

  const total = first.total
  const items: Task[] = [...first.items]
  if (items.length >= total) return { items, total }

  // Оставшиеся страницы грузим параллельно.
  const remaining = total - items.length
  const pages = Math.ceil(remaining / FETCH_LIMIT)
  const promises: Promise<TasksResponse>[] = []

  for (let p = 1; p < pages; p++) {
    const offset = p * FETCH_LIMIT
    const limit = Math.min(FETCH_LIMIT, total - offset)
    promises.push(
      api.listTasks({ team_id: teamId, assignee_id: assigneeId, limit, offset }),
    )
  }

  const responses = await Promise.all(promises)
  if (signal?.aborted) return { items: [], total: 0 }

  for (const res of responses) {
    items.push(...res.items)
  }

  return { items, total }
}

export function TeamPage() {
  const { teamId: teamIdParam } = useParams()
  const teamId = Number(teamIdParam)

  const [team, setTeam] = useState<TeamWithRole | null>(null)
  const [tasks, setTasks] = useState<Task[]>([])
  const [total, setTotal] = useState(0)
  const [assigneeFilter, setAssigneeFilter] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<unknown>(null)

  const [view, setView] = useState<View>('board')
  const [selected, setSelected] = useState<Task | null>(null)

  const [showCreate, setShowCreate] = useState(false)
  const [createTitle, setCreateTitle] = useState('')
  const [createDesc, setCreateDesc] = useState('')
  const [createStatus, setCreateStatus] = useState<TaskStatus>('todo')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<unknown>(null)

  const [showInvite, setShowInvite] = useState(false)
  const [inviteUserId, setInviteUserId] = useState('')
  const [inviteRole, setInviteRole] = useState<InviteRole>('member')
  const [inviting, setInviting] = useState(false)
  const [inviteError, setInviteError] = useState<unknown>(null)
  const [inviteOk, setInviteOk] = useState('')

  const [stats, setStats] = useState<TeamStatsResponse | null>(null)
  const [statsLoading, setStatsLoading] = useState(false)
  const [statsError, setStatsError] = useState<unknown>(null)

  const [teamMembers, setTeamMembers] = useState<TeamMemberInfo[]>([])
  const [membersLoading, setMembersLoading] = useState(false)

  // AbortController для отмены предыдущих запросов (Strict Mode)
  const loadRef = useRef<{ controller: AbortController | null }>({
    controller: null,
  })

  const canManage = team?.role === 'owner' || team?.role === 'admin'
  const inviteRoles: InviteRole[] =
    team?.role === 'owner' ? ['admin', 'member'] : ['member']

  const loadTeam = useCallback(async () => {
    const res = await api.listTeams()
    const found = res.items.find((t) => t.id === teamId) ?? null
    setTeam(found)
    return found
  }, [teamId])

  const loadTasks = useCallback(async () => {
    // Отменяем предыдущий запрос
    loadRef.current.controller?.abort()
    const controller = new AbortController()
    loadRef.current.controller = controller

    const assignee =
      assigneeFilter.trim() === ''
        ? undefined
        : Number.parseInt(assigneeFilter, 10)

    setLoading(true)
    setError(null)
    try {
      const res = await fetchAllTasks(
        teamId,
        assignee != null && !Number.isNaN(assignee) ? assignee : undefined,
        controller.signal,
      )
      if (controller.signal.aborted) return

      setTasks(res.items)
      setTotal(res.total)
      setSelected((cur) => {
        if (!cur) return null
        return res.items.find((t) => t.id === cur.id) ?? null
      })
    } catch (err) {
      if (!controller.signal.aborted) setError(err)
    } finally {
      if (!controller.signal.aborted) setLoading(false)
    }
  }, [teamId, assigneeFilter])

  useEffect(() => {
    if (!Number.isFinite(teamId) || teamId < 1) {
      setError(new Error('Некорректный идентификатор команды'))
      setLoading(false)
      return
    }
    const controller = loadRef.current.controller
    void (async () => {
      try {
        await Promise.all([loadTeam(), loadTasks()])
      } catch (err) {
        setError(err)
        setLoading(false)
      }
    })()
    return () => {
      controller?.abort()
    }
  }, [teamId, loadTeam, loadTasks])

  useEffect(() => {
    if (view !== 'stats' || !canManage) return
    let cancelled = false
    setStatsLoading(true)
    setStatsError(null)
    api
      .getTeamStats(teamId)
      .then((res) => {
        if (!cancelled) setStats(res)
      })
      .catch((err) => {
        if (!cancelled) setStatsError(err)
      })
      .finally(() => {
        if (!cancelled) setStatsLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [view, canManage, teamId])

  useEffect(() => {
    let cancelled = false
    setMembersLoading(true)
    api
      .listTeamMembers(teamId)
      .then((res) => {
        if (!cancelled) setTeamMembers(res.items)
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

  async function onCreateTask(e: FormEvent) {
    e.preventDefault()
    setCreating(true)
    setCreateError(null)
    try {
      const created = await api.createTask({
        team_id: teamId,
        title: createTitle.trim(),
        description: createDesc,
        status: createStatus,
      })
      setShowCreate(false)
      setCreateTitle('')
      setCreateDesc('')
      setCreateStatus('todo')
      setTasks((prev) => [created, ...prev])
      setTotal((t) => t + 1)
      setSelected(created)
      setView('board')
    } catch (err) {
      setCreateError(err)
    } finally {
      setCreating(false)
    }
  }

  async function onInvite(e: FormEvent) {
    e.preventDefault()
    setInviting(true)
    setInviteError(null)
    setInviteOk('')
    try {
      const userId = Number.parseInt(inviteUserId, 10)
      await api.inviteMember(teamId, userId, inviteRole)
      setInviteOk(
        `Пользователь #${userId} приглашён как ${ROLE_LABELS[inviteRole]}`,
      )
      setInviteUserId('')
    } catch (err) {
      setInviteError(err)
    } finally {
      setInviting(false)
    }
  }

  function onTaskUpdated(updated: Task) {
    setTasks((prev) => prev.map((t) => (t.id === updated.id ? updated : t)))
    setSelected(updated)
  }

  if (!Number.isFinite(teamId) || teamId < 1) {
    return (
      <div className="page">
        <ErrorAlert error={error} />
        <Link to="/">← К командам</Link>
      </div>
    )
  }

  return (
    <div className={`page team-page ${selected ? 'with-panel' : ''}`}>
      <div className="team-main">
        <div className="page-header">
          <div>
            <Link to="/" className="back-link">
              ← Команды
            </Link>
            <h1>{team?.name ?? `Команда #${teamId}`}</h1>
            {team && (
              <p className="muted">
                Ваша роль: {ROLE_LABELS[team.role as Role]}
              </p>
            )}
          </div>
          <div className="header-actions">
            {canManage && (
              <button
                type="button"
                className="btn btn-ghost"
                onClick={() => {
                  setInviteError(null)
                  setInviteOk('')
                  setShowInvite(true)
                }}
              >
                Пригласить
              </button>
            )}
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => setShowCreate(true)}
            >
              Новая задача
            </button>
          </div>
        </div>

        <div className="tabs page-tabs">
          <button
            type="button"
            className={`tab ${view === 'board' ? 'active' : ''}`}
            onClick={() => setView('board')}
          >
            Доска
          </button>
          {canManage && (
            <button
              type="button"
              className={`tab ${view === 'stats' ? 'active' : ''}`}
              onClick={() => setView('stats')}
            >
              Статистика
            </button>
          )}
        </div>

        <ErrorAlert error={error} />

        {view === 'board' && (
          <>
            <div className="filters">
              <label>
                Исполнитель
                <select
                  value={assigneeFilter}
                  onChange={(e) => setAssigneeFilter(e.target.value)}
                  disabled={membersLoading}
                >
                  <option value="">Любой</option>
                  {teamMembers.map((m) => (
                    <option key={m.user_id} value={String(m.user_id)}>
                      {m.name} (#{m.user_id})
                    </option>
                  ))}
                </select>
              </label>
              <button
                type="button"
                className="btn btn-ghost"
                onClick={() => void loadTasks()}
                disabled={loading || membersLoading}
              >
                Обновить
              </button>
              <span className="filters-meta">
                {loading ? 'Загрузка…' : `${total} задач на доске`}
              </span>
            </div>

            {!loading && tasks.length === 0 ? (
              <div className="empty">
                <h2>Доска пуста</h2>
                <p className="muted">
                  Создайте задачу или добавьте карточку прямо в колонке.
                </p>
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={() => setShowCreate(true)}
                >
                  Создать задачу
                </button>
              </div>
            ) : (
              <KanbanBoard
                teamId={teamId}
                tasks={tasks}
                selectedId={selected?.id ?? null}
                busy={loading}
                onSelect={setSelected}
                onTasksChange={setTasks}
                onCreated={(task) => {
                  setSelected(task)
                  setTotal((t) => t + 1)
                }}
                onError={setError}
              />
            )}
          </>
        )}

        {view === 'stats' && (
          <div className="stats">
            <ErrorAlert error={statsError} />
            {statsLoading ? (
              <p className="muted">Загрузка…</p>
            ) : stats ? (
              <>
                <div className="stats-grid">
                  {(Object.keys(stats.tasks_by_status) as TaskStatus[]).map(
                    (key) => (
                      <div key={key} className="stat-tile">
                        <span className="stat-label">{STATUS_LABELS[key]}</span>
                        <span className="stat-value">
                          {stats.tasks_by_status[key]}
                        </span>
                      </div>
                    ),
                  )}
                  <div className="stat-tile">
                    <span className="stat-label">Комментарии</span>
                    <span className="stat-value">{stats.comments_count}</span>
                  </div>
                  <div className="stat-tile">
                    <span className="stat-label">Среднее закрытие</span>
                    <span className="stat-value">
                      {formatDuration(stats.average_close_seconds)}
                    </span>
                  </div>
                </div>

                <h2 className="section-title">Топ исполнителей (30 дней)</h2>
                {stats.top_assignees.length === 0 ? (
                  <p className="muted">Пока нет закрытых задач за период.</p>
                ) : (
                  <ul className="assignee-list">
                    {stats.top_assignees.map((a) => (
                      <li key={a.user_id}>
                        <span>
                          {a.name} <span className="muted">#{a.user_id}</span>
                        </span>
                        <strong>{a.closed_tasks}</strong>
                      </li>
                    ))}
                  </ul>
                )}
              </>
            ) : null}
          </div>
        )}
      </div>

      {selected && team && (
        <TaskPanel
          task={selected}
          role={team.role}
          teamId={teamId}
          onClose={() => setSelected(null)}
          onUpdated={onTaskUpdated}
        />
      )}

      {showCreate && (
        <Modal title="Новая задача" onClose={() => setShowCreate(false)}>
          <ErrorAlert error={createError} />
          <form className="form" onSubmit={onCreateTask}>
            <label>
              Заголовок
              <input
                required
                maxLength={255}
                autoFocus
                value={createTitle}
                onChange={(e) => setCreateTitle(e.target.value)}
              />
            </label>
            <label>
              Описание
              <textarea
                rows={4}
                maxLength={5000}
                value={createDesc}
                onChange={(e) => setCreateDesc(e.target.value)}
              />
            </label>
            <label>
              Колонка
              <select
                value={createStatus}
                onChange={(e) => setCreateStatus(e.target.value as TaskStatus)}
              >
                {TASK_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {STATUS_LABELS[s]}
                  </option>
                ))}
              </select>
            </label>
            <div className="form-actions">
              <button
                type="button"
                className="btn btn-ghost"
                onClick={() => setShowCreate(false)}
              >
                Отмена
              </button>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={creating}
              >
                {creating ? 'Создаём…' : 'Создать'}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {showInvite && (
        <Modal title="Пригласить участника" onClose={() => setShowInvite(false)}>
          <ErrorAlert error={inviteError} />
          {inviteOk && <div className="alert alert-ok">{inviteOk}</div>}
          <p className="muted">
            API принимает числовой <code>user_id</code> зарегистрированного
            пользователя.
          </p>
          <form className="form" onSubmit={onInvite}>
            <label>
              User ID
              <input
                type="number"
                min={1}
                required
                value={inviteUserId}
                onChange={(e) => setInviteUserId(e.target.value)}
              />
            </label>
            <label>
              Роль
              <select
                value={inviteRole}
                onChange={(e) => setInviteRole(e.target.value as InviteRole)}
              >
                {inviteRoles.map((r) => (
                  <option key={r} value={r}>
                    {ROLE_LABELS[r]}
                  </option>
                ))}
              </select>
            </label>
            <div className="form-actions">
              <button
                type="button"
                className="btn btn-ghost"
                onClick={() => setShowInvite(false)}
              >
                Закрыть
              </button>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={inviting}
              >
                {inviting ? 'Приглашаем…' : 'Пригласить'}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
