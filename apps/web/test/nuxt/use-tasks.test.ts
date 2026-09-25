import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/services/client'
import type { Task, TaskStatus } from '~/types/api'

let nextId = 1
function task(title: string, completed = false): Task {
  const at = '2026-09-25T12:00:00Z'
  return { id: nextId++, title, completed, createdAt: at, updatedAt: at }
}

function api() {
  return useNuxtApp().$api.tasks
}

/** A promise the test resolves or rejects by hand, to observe in-flight state. */
function deferred<T>() {
  let resolve!: (v: T) => void
  let reject!: (e: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

async function setup(initial: Task[], status = ref<TaskStatus>('all')) {
  vi.spyOn(api(), 'list').mockResolvedValue(initial)
  const state = useTasks(status)
  await flushPromises()
  return { state, status }
}

describe('useTasks', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('loads the list for the filter and reloads when it changes', async () => {
    const pending = task('pending')
    const done = task('done', true)
    const list = vi
      .spyOn(api(), 'list')
      .mockImplementation(async (s) =>
        s === 'completed' ? [done] : [pending, done],
      )
    const status = ref<TaskStatus>('all')
    const { tasks, loading } = useTasks(status)

    expect(loading.value).toBe(true)
    await flushPromises()
    expect(loading.value).toBe(false)
    expect(tasks.value).toEqual([pending, done])

    status.value = 'completed'
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith('completed')
    expect(tasks.value).toEqual([done])
  })

  it('keeps only the newest load when filters change quickly', async () => {
    const slow = deferred<Task[]>()
    const fresh = [task('fresh')]
    vi.spyOn(api(), 'list')
      .mockReturnValueOnce(slow.promise)
      .mockResolvedValueOnce(fresh)
    const status = ref<TaskStatus>('all')
    const { tasks } = useTasks(status)

    status.value = 'pending'
    await flushPromises()
    slow.resolve([task('stale')])
    await flushPromises()

    expect(tasks.value).toEqual(fresh)
  })

  it('reports a load failure', async () => {
    vi.spyOn(api(), 'list').mockRejectedValue(
      new ApiError(0, 'NETWORK_ERROR', 'offline'),
    )
    const { loadError, loading } = useTasks(ref<TaskStatus>('all'))
    await flushPromises()

    expect(loadError.value?.code).toBe('NETWORK_ERROR')
    expect(loading.value).toBe(false)
  })

  it('adds a created task at the top, unless the filter hides it', async () => {
    const old = task('old')
    const { state, status } = await setup([old])
    const created = task('new')
    vi.spyOn(api(), 'create').mockResolvedValue(created)

    await state.create('new')
    expect(state.tasks.value).toEqual([created, old])

    status.value = 'completed'
    vi.spyOn(api(), 'list').mockResolvedValue([])
    await flushPromises()
    await state.create('another')
    expect(state.tasks.value).toEqual([])
  })

  it('throws create failures to the caller', async () => {
    const { state } = await setup([])
    vi.spyOn(api(), 'create').mockRejectedValue(
      new ApiError(400, 'VALIDATION_FAILED', '', {
        title: 'Title is required',
      }),
    )

    await expect(state.create(' ')).rejects.toMatchObject({
      code: 'VALIDATION_FAILED',
    })
    expect(state.tasks.value).toEqual([])
  })

  it('marks a task pending while its update runs, then applies it', async () => {
    const t = task('Buy milk')
    const { state } = await setup([t])
    const reply = deferred<Task>()
    vi.spyOn(api(), 'update').mockReturnValue(reply.promise)

    const result = state.update(t.id, { title: 'Buy oat milk' })
    expect(state.isPending(t.id)).toBe(true)
    expect(state.tasks.value[0]!.title).toBe('Buy milk')

    reply.resolve({ ...t, title: 'Buy oat milk' })
    expect(await result).toBe(true)
    expect(state.isPending(t.id)).toBe(false)
    expect(state.tasks.value[0]!.title).toBe('Buy oat milk')
  })

  it('drops a task that no longer matches the filter', async () => {
    const t = task('Buy milk')
    const { state } = await setup([t], ref<TaskStatus>('pending'))
    vi.spyOn(api(), 'update').mockResolvedValue({ ...t, completed: true })

    await state.update(t.id, { completed: true })

    expect(state.tasks.value).toEqual([])
  })

  it('keeps the list and reports the error when an update fails', async () => {
    const t = task('Buy milk')
    const { state } = await setup([t])
    vi.spyOn(api(), 'update').mockRejectedValue(
      new ApiError(0, 'NETWORK_ERROR', 'offline'),
    )

    expect(await state.update(t.id, { completed: true })).toBe(false)

    expect(state.tasks.value).toEqual([t])
    expect(state.actionError.value?.code).toBe('NETWORK_ERROR')
    expect(state.isPending(t.id)).toBe(false)
  })

  it('removes a deleted task, and one already gone elsewhere', async () => {
    const a = task('a')
    const b = task('b')
    const { state } = await setup([a, b])
    const remove = vi.spyOn(api(), 'remove').mockResolvedValueOnce(undefined)

    expect(await state.remove(a.id)).toBe(true)
    expect(state.tasks.value).toEqual([b])

    remove.mockRejectedValueOnce(
      new ApiError(404, 'TASK_NOT_FOUND', 'Task not found'),
    )
    expect(await state.remove(b.id)).toBe(false)
    expect(state.tasks.value).toEqual([])
    expect(state.actionError.value?.code).toBe('TASK_NOT_FOUND')
  })
})
