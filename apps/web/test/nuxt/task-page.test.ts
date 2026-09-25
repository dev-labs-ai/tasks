import { mountSuspended } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import IndexPage from '~/pages/index.vue'
import { ApiError } from '~/services/client'
import type { Task } from '~/types/api'

function task(id: number, title: string): Task {
  const at = '2026-09-25T12:00:00Z'
  return { id, title, completed: false, createdAt: at, updatedAt: at }
}

async function mountPage() {
  const wrapper = await mountSuspended(IndexPage, {
    route: '/',
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

describe('task page', () => {
  beforeEach(() => {
    useAuthStore().setUser({ id: 1, email: 'ana@example.com', createdAt: '' })
  })

  afterEach(() => {
    vi.restoreAllMocks()
    document.body.innerHTML = ''
  })

  it('shows skeletons while loading', async () => {
    vi.spyOn(useNuxtApp().$api.tasks, 'list').mockReturnValue(
      new Promise(() => {}),
    )

    const wrapper = await mountPage()

    expect(wrapper.find('[aria-busy="true"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-slot="skeleton"]')).toHaveLength(3)
  })

  it('shows an empty-state message', async () => {
    vi.spyOn(useNuxtApp().$api.tasks, 'list').mockResolvedValue([])

    const wrapper = await mountPage()

    expect(wrapper.get('[data-testid="empty"]').text()).toBe(
      'No tasks yet. Add your first one above.',
    )
  })

  it('shows a load error with a retry', async () => {
    const list = vi
      .spyOn(useNuxtApp().$api.tasks, 'list')
      .mockRejectedValueOnce(
        new ApiError(0, 'NETWORK_ERROR', 'Could not reach the server.'),
      )
      .mockResolvedValueOnce([task(1, 'Buy milk')])

    const wrapper = await mountPage()
    const alert = wrapper.get('[data-testid="load-error"]')
    expect(alert.attributes('role')).toBe('alert')
    expect(alert.text()).toContain('Could not reach the server.')

    await alert.get('button').trigger('click')
    await flushPromises()

    expect(list).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('Buy milk')
  })

  it('moves focus to the next task after a delete, then to the input', async () => {
    vi.spyOn(useNuxtApp().$api.tasks, 'list').mockResolvedValue([
      task(1, 'first'),
      task(2, 'second'),
    ])
    vi.spyOn(useNuxtApp().$api.tasks, 'remove').mockResolvedValue(undefined)
    const wrapper = await mountPage()

    await wrapper.get('[aria-label="Delete task: first"]').trigger('click')
    await flushPromises()
    expect(document.activeElement?.getAttribute('aria-label')).toBe('second')

    await wrapper.get('[aria-label="Delete task: second"]').trigger('click')
    await flushPromises()
    expect(document.activeElement).toBe(wrapper.get('form input').element)
  })

  it('reports a failed change above the list', async () => {
    vi.spyOn(useNuxtApp().$api.tasks, 'list').mockResolvedValue([
      task(1, 'Buy milk'),
    ])
    vi.spyOn(useNuxtApp().$api.tasks, 'update').mockRejectedValue(
      new ApiError(500, 'INTERNAL_ERROR', 'Something went wrong'),
    )
    const wrapper = await mountPage()

    await wrapper.get('[data-slot="checkbox"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="action-error"]').text()).toBe(
      'Could not save the change. Please try again.',
    )
  })
})
