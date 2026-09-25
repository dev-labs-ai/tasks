import type { $Fetch } from 'ofetch'
import type { Task, TaskStatus } from '~/types/api'
import { call } from './client'

export interface TaskPatch {
  title?: string
  completed?: boolean
}

export function tasksService(client: $Fetch) {
  return {
    list: (status: TaskStatus) =>
      call<Task[]>(client, '/tasks', {
        query: status === 'all' ? {} : { status },
      }),
    create: (title: string) =>
      call<Task>(client, '/tasks', { method: 'POST', body: { title } }),
    update: (id: number, patch: TaskPatch) =>
      call<Task>(client, `/tasks/${id}`, { method: 'PATCH', body: patch }),
    remove: (id: number) =>
      call<undefined>(client, `/tasks/${id}`, { method: 'DELETE' }),
  }
}

export type TasksService = ReturnType<typeof tasksService>
