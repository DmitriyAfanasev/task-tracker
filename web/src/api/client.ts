import type {
  AuthResponse,
  InviteRole,
  Task,
  TaskComment,
  TaskCommentsResponse,
  TaskHistoryResponse,
  TaskStatus,
  TasksResponse,
  Team,
  TeamMember,
  TeamMembersResponse,
  TeamStatsResponse,
  TeamsResponse,
} from './types'
import { ApiError, type ErrorResponse } from './types'

const BASE = '/api/v1'

type TokenProvider = () => string | null

let getToken: TokenProvider = () => null
let onUnauthorized: (() => void) | null = null

export function configureClient(options: {
  getToken: TokenProvider
  onUnauthorized?: () => void
}) {
  getToken = options.getToken
  onUnauthorized = options.onUnauthorized ?? null
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  auth = true,
): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (auth) {
    const token = getToken()
    if (token) headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch(`${BASE}${path}`, { ...init, headers })

  if (res.status === 204) {
    return undefined as T
  }

  const text = await res.text()
  const data = text ? (JSON.parse(text) as unknown) : null

  if (!res.ok) {
    const err = (data as ErrorResponse | null)?.error ?? {
      code: 'unknown',
      message: res.statusText || 'Ошибка запроса',
      request_id: '',
    }
    if (res.status === 401 && auth) onUnauthorized?.()
    throw new ApiError(res.status, err)
  }

  return data as T
}

export const api = {
  register(body: { email: string; password: string; name: string }) {
    return request<AuthResponse>('/register', {
      method: 'POST',
      body: JSON.stringify(body),
    }, false)
  },

  login(body: { email: string; password: string }) {
    return request<AuthResponse>('/login', {
      method: 'POST',
      body: JSON.stringify(body),
    }, false)
  },

  listTeams() {
    return request<TeamsResponse>('/teams')
  },

  createTeam(name: string) {
    return request<Team>('/teams', {
      method: 'POST',
      body: JSON.stringify({ name }),
    })
  },

  inviteMember(teamId: number, userId: number, role: InviteRole) {
    return request<TeamMember>(`/teams/${teamId}/invite`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId, role }),
    })
  },

  getTeamStats(teamId: number) {
    return request<TeamStatsResponse>(`/teams/${teamId}/stats`)
  },

  listTeamMembers(teamId: number) {
    return request<TeamMembersResponse>(`/teams/${teamId}/members`)
  },

  listTasks(params: {
    team_id: number
    status?: TaskStatus
    assignee_id?: number
    limit?: number
    offset?: number
  }) {
    const q = new URLSearchParams()
    q.set('team_id', String(params.team_id))
    if (params.status) q.set('status', params.status)
    if (params.assignee_id) q.set('assignee_id', String(params.assignee_id))
    if (params.limit != null) q.set('limit', String(params.limit))
    if (params.offset != null) q.set('offset', String(params.offset))
    return request<TasksResponse>(`/tasks?${q}`)
  },

  createTask(body: {
    team_id: number
    title: string
    description?: string
    status?: TaskStatus
  }) {
    return request<Task>('/tasks', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },

  updateTask(
    id: number,
    body: {
      version: number
      title?: string
      description?: string
      status?: TaskStatus
      assignee_id?: number
    },
  ) {
    return request<Task>(`/tasks/${id}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    })
  },

  listTaskHistory(id: number) {
    return request<TaskHistoryResponse>(`/tasks/${id}/history`)
  },

  listTaskComments(id: number, limit = 50, offset = 0) {
    const q = new URLSearchParams({
      limit: String(limit),
      offset: String(offset),
    })
    return request<TaskCommentsResponse>(`/tasks/${id}/comments?${q}`)
  },

  addTaskComment(id: number, content: string) {
    return request<TaskComment>(`/tasks/${id}/comments`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    })
  },
}
