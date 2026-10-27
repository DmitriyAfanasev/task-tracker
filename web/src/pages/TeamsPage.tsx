import { useEffect, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { ROLE_LABELS, type TeamWithRole } from '../api/types'
import { ErrorAlert } from '../components/ErrorAlert'
import { Modal } from '../components/Modal'
import { formatDate } from '../lib/format'

export function TeamsPage() {
  const [teams, setTeams] = useState<TeamWithRole[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<unknown>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [name, setName] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<unknown>(null)

  async function load() {
    setLoading(true)
    setError(null)
    try {
      const res = await api.listTeams()
      setTeams(res.items)
    } catch (err) {
      setError(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    setCreating(true)
    setCreateError(null)
    try {
      await api.createTeam(name.trim())
      setName('')
      setShowCreate(false)
      await load()
    } catch (err) {
      setCreateError(err)
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1>Команды</h1>
          <p className="muted">Выберите команду или создайте новую.</p>
        </div>
        <button
          type="button"
          className="btn btn-primary"
          onClick={() => setShowCreate(true)}
        >
          Новая команда
        </button>
      </div>

      <ErrorAlert error={error} />

      {loading ? (
        <p className="muted">Загрузка…</p>
      ) : teams.length === 0 ? (
        <div className="empty">
          <h2>Пока нет команд</h2>
          <p className="muted">Создайте первую команду, чтобы начать работу с задачами.</p>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => setShowCreate(true)}
          >
            Создать команду
          </button>
        </div>
      ) : (
        <ul className="team-list">
          {teams.map((team) => (
            <li key={team.id}>
              <Link to={`/teams/${team.id}`} className="team-card">
                <div>
                  <h2>{team.name}</h2>
                  <p className="muted">
                    Создана {formatDate(team.created_at)}
                  </p>
                </div>
                <span className="role-pill">{ROLE_LABELS[team.role]}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}

      {showCreate && (
        <Modal title="Новая команда" onClose={() => setShowCreate(false)}>
          <ErrorAlert error={createError} />
          <form className="form" onSubmit={onCreate}>
            <label>
              Название
              <input
                type="text"
                required
                maxLength={100}
                autoFocus
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Платформа"
              />
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
    </div>
  )
}
