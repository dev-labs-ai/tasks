// Shapes of the Go API's JSON (tech spec §6). Timestamps are RFC 3339 strings.

export interface User {
  id: number
  email: string
  createdAt: string
}

export interface Task {
  id: number
  title: string
  completed: boolean
  createdAt: string
  updatedAt: string
}

/** The list filter. `all` is sent as no `status` parameter. */
export type TaskStatus = 'all' | 'pending' | 'completed'

export type ErrorCode =
  | 'MALFORMED_REQUEST'
  | 'VALIDATION_FAILED'
  | 'UNAUTHENTICATED'
  | 'INVALID_CREDENTIALS'
  | 'FORBIDDEN_ORIGIN'
  | 'TASK_NOT_FOUND'
  | 'NOT_FOUND'
  | 'EMAIL_TAKEN'
  | 'PAYLOAD_TOO_LARGE'
  | 'TOO_MANY_ATTEMPTS'
  | 'INTERNAL_ERROR'

export interface ErrorBody {
  error: {
    code: ErrorCode
    message: string
    fields?: Record<string, string>
  }
}

export interface DataBody<T> {
  data: T
}
