import { flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { parseTaskStatus } from '~/composables/useTaskFilter'

describe('parseTaskStatus', () => {
  it.each([
    ['pending', 'pending'],
    ['completed', 'completed'],
    [undefined, 'all'],
    [null, 'all'],
    ['', 'all'],
    ['done', 'all'],
    ['Pending', 'all'],
    [['completed', 'pending'], 'completed'],
  ] as const)('%j reads as %j', (value, want) => {
    expect(parseTaskStatus(value as never)).toBe(want)
  })
})

describe('useTaskFilter', () => {
  beforeEach(() => {
    // A signed-in user, so the route guard lets the task page through.
    const auth = useAuthStore()
    auth.setUser({ id: 1, email: 'ana@example.com', createdAt: '' })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('reads the filter from the URL', async () => {
    await useRouter().push('/?status=completed')

    expect(useTaskFilter().value).toBe('completed')
  })

  it('treats an unknown value as all', async () => {
    await useRouter().push('/?status=everything')

    expect(useTaskFilter().value).toBe('all')
  })

  it('writes the filter to the URL, and drops it for all', async () => {
    const router = useRouter()
    await router.push('/')
    const filter = useTaskFilter()

    filter.value = 'pending'
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/?status=pending')

    filter.value = 'all'
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/')
  })
})
