export type Role = 'owner' | 'admin' | 'member'
export type InviteRole = 'admin' | 'member'
export type TaskStatus = 'todo' | 'in_progress' | 'done' | 'cancelled'

export interface User {
  id: number
  email: string
  name: string
  created_at: string
}

export interface AuthResponse {
  token: string
  expires_at: string
  user: User
}

export interface Team {
  id: number
  name: string
  created_by: number
  created_at: string
}

export interface TeamWithRole extends Team {
  role: Role
}

export interface TeamsResponse {
  items: TeamWithRole[]
}

export interface TeamMember {
  team_id: number
  user_id: number
  role: Role
}

export interface TeamMemberInfo {
  user_id: number
  name: string
  email: string
  role: Role
}

export interface TeamMembersResponse {
  items: TeamMemberInfo[]
}

export interface Task {
  id: number
  team_id: number
  title: string
  description: string
  status: TaskStatus
  created_by: number
  assignee_id: number | null
  created_at: string
  updated_at: string
  closed_at: string | null
  version: number
}

export interface TasksResponse {
  items: Task[]
  total: number
  limit: number
  offset: number
}

export interface TaskChange {
  field: string
  old: unknown
  new: unknown
}

export interface TaskHistoryEntry {
  id: number
  task_id: number
  changed_by: number
  changes: TaskChange[]
  created_at: string
}

export interface TaskHistoryResponse {
  items: TaskHistoryEntry[]
}

export interface TaskComment {
  id: number
  task_id: number
  user_id: number
  content: string
  created_at: string
}

export interface TaskCommentsResponse {
  items: TaskComment[]
  total: number
  limit: number
  offset: number
}

export interface TasksByStatus {
  todo: number
  in_progress: number
  done: number
  cancelled: number
}

export interface TopAssignee {
  user_id: number
  name: string
  closed_tasks: number
}

export interface TeamStatsResponse {
  team_id: number
  tasks_by_status: TasksByStatus
  top_assignees: TopAssignee[]
  average_close_seconds: number | null
  comments_count: number
}

export interface ErrorBody {
  code: string
  message: string
  request_id: string
  details?: Record<string, string>
}

export interface ErrorResponse {
  error: ErrorBody
}

export class ApiError extends Error {
  readonly status: number
  readonly body: ErrorBody

  constructor(status: number, body: ErrorBody) {
    super(body.message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

export const TASK_STATUSES: TaskStatus[] = [
  'todo',
  'in_progress',
  'done',
  'cancelled',
]

export const STATUS_LABELS: Record<TaskStatus, string> = {
  todo: 'К выполнению',
  in_progress: 'В работе',
  done: 'Готово',
  cancelled: 'Отменено',
}

export const ROLE_LABELS: Record<Role, string> = {
  owner: 'Владелец',
  admin: 'Админ',
  member: 'Участник',
}
