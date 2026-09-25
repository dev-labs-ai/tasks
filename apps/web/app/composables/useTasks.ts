import type { MaybeRefOrGetter } from 'vue'
import { toApiError, type ApiError } from '~/services/client'
import type { TaskPatch } from '~/services/tasks'
import type { Task, TaskStatus } from '~/types/api'

/**
 * The task list for the current filter (tech spec §7.4). Updates are not
 * optimistic: local state changes only after the API confirms.
 */
export function useTasks(status: MaybeRefOrGetter<TaskStatus>) {
  const { $api } = useNuxtApp()

  const tasks = ref<Task[]>([])
  const loading = ref(false)
  /** Why the list could not load; the page shows it instead of the list. */
  const loadError = ref<ApiError | null>(null)
  /** Why the last update or delete failed; shown above the list. */
  const actionError = ref<ApiError | null>(null)
  const pendingIds = ref(new Set<number>())

  // Only the newest load may write the list, so a slow response for an old
  // filter cannot replace the current one.
  let loadCount = 0

  async function load() {
    const current = ++loadCount
    loading.value = true
    loadError.value = null
    try {
      const result = await $api.tasks.list(toValue(status))
      if (current === loadCount) tasks.value = result
    } catch (error) {
      if (current === loadCount) loadError.value = toApiError(error)
    } finally {
      if (current === loadCount) loading.value = false
    }
  }

  function matchesFilter(task: Task): boolean {
    const filter = toValue(status)
    return filter === 'all' || (filter === 'completed') === task.completed
  }

  /** Creates a task. Throws ApiError, so the form can keep the text and show why. */
  async function create(title: string): Promise<Task> {
    try {
      const task = await $api.tasks.create(title)
      if (matchesFilter(task)) tasks.value = [task, ...tasks.value]
      return task
    } catch (error) {
      throw toApiError(error)
    }
  }

  async function run(
    id: number,
    action: () => Promise<void>,
  ): Promise<boolean> {
    actionError.value = null
    pendingIds.value.add(id)
    try {
      await action()
      return true
    } catch (error) {
      const apiError = toApiError(error)
      // Deleted in another tab or device: drop it here too.
      if (apiError.code === 'TASK_NOT_FOUND') drop(id)
      actionError.value = apiError
      return false
    } finally {
      pendingIds.value.delete(id)
    }
  }

  function drop(id: number) {
    tasks.value = tasks.value.filter((t) => t.id !== id)
  }

  /** Changes a task; one that no longer matches the filter leaves the list. */
  function update(id: number, patch: TaskPatch): Promise<boolean> {
    return run(id, async () => {
      const updated = await $api.tasks.update(id, patch)
      tasks.value = matchesFilter(updated)
        ? tasks.value.map((t) => (t.id === id ? updated : t))
        : tasks.value.filter((t) => t.id !== id)
    })
  }

  function remove(id: number): Promise<boolean> {
    return run(id, async () => {
      await $api.tasks.remove(id)
      drop(id)
    })
  }

  function isPending(id: number): boolean {
    return pendingIds.value.has(id)
  }

  watch(() => toValue(status), load, { immediate: true })

  return {
    tasks: readonly(tasks),
    loading: readonly(loading),
    loadError: readonly(loadError),
    actionError,
    load,
    create,
    update,
    remove,
    isPending,
  }
}
