import type { LocationQueryValue } from 'vue-router'
import type { TaskStatus } from '~/types/api'

/** Reads ?status=; anything but pending or completed means all (PRD FR-10). */
export function parseTaskStatus(
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): TaskStatus {
  const first = Array.isArray(value) ? value[0] : value
  return first === 'pending' || first === 'completed' ? first : 'all'
}

/**
 * The task filter, kept in the URL so it survives a reload. Setting it
 * replaces the history entry; "all" drops the parameter.
 */
export function useTaskFilter() {
  const route = useRoute()
  const router = useRouter()

  return computed<TaskStatus>({
    get: () => parseTaskStatus(route.query.status),
    set: (status) => {
      router.replace({
        query: {
          ...route.query,
          status: status === 'all' ? undefined : status,
        },
      })
    },
  })
}
