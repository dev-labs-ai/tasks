import { mountSuspended } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import TaskFilter from '~/components/tasks/TaskFilter.vue'
import TaskForm from '~/components/tasks/TaskForm.vue'
import TaskItem from '~/components/tasks/TaskItem.vue'
import TaskList from '~/components/tasks/TaskList.vue'
import { ApiError } from '~/services/client'
import type { Task } from '~/types/api'

function task(id: number, title: string, completed = false): Task {
  const at = '2026-09-25T12:00:00Z'
  return { id, title, completed, createdAt: at, updatedAt: at }
}

function inputValue(
  wrapper: { find: (s: string) => { element: Element } },
  selector = 'input',
) {
  return (wrapper.find(selector).element as HTMLInputElement).value
}

describe('TaskForm', () => {
  async function mountForm(create: (title: string) => Promise<Task>) {
    return mountSuspended(TaskForm, { props: { create } })
  }

  it('submits the trimmed title and clears the input', async () => {
    const create = vi.fn(async (title: string) => task(1, title))
    const wrapper = await mountForm(create)

    await wrapper.find('input').setValue('  Buy milk ')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(create).toHaveBeenCalledWith('Buy milk')
    expect(inputValue(wrapper)).toBe('')
  })

  it('checks the title before calling the API', async () => {
    const create = vi.fn()
    const wrapper = await mountForm(create)

    await wrapper.find('input').setValue('   ')
    await wrapper.find('form').trigger('submit')

    expect(create).not.toHaveBeenCalled()
    expect(wrapper.get('[role="alert"]').text()).toBe('Title is required')
  })

  it('keeps the text and shows why when the API refuses', async () => {
    const create = vi
      .fn()
      .mockRejectedValue(
        new ApiError(0, 'NETWORK_ERROR', 'Could not reach the server.'),
      )
    const wrapper = await mountForm(create)

    await wrapper.find('input').setValue('Buy milk')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(inputValue(wrapper)).toBe('Buy milk')
    expect(wrapper.get('[role="alert"]').text()).toBe(
      'Could not reach the server.',
    )
    expect(wrapper.find('input').attributes('aria-invalid')).toBe('true')
  })

  it('has an accessible name for its input', async () => {
    const wrapper = await mountForm(vi.fn())
    const id = wrapper.find('input').attributes('id')

    expect(wrapper.find(`label[for="${id}"]`).text()).toBe('New task')
  })
})

describe('TaskItem', () => {
  async function mountItem(
    t: Task,
    overrides: Partial<{
      pending: boolean
      update: unknown
      remove: unknown
    }> = {},
  ) {
    const update = vi.fn(async () => true)
    const remove = vi.fn(async () => true)
    const wrapper = await mountSuspended(TaskItem, {
      props: { task: t, pending: false, update, remove, ...overrides } as never,
    })
    return { wrapper, update, remove }
  }

  it('toggles completion through the checkbox', async () => {
    const { wrapper, update } = await mountItem(task(1, 'Buy milk'))

    await wrapper.get('[data-slot="checkbox"]').trigger('click')

    expect(update).toHaveBeenCalledWith({ completed: true })
  })

  it('names its controls after the task', async () => {
    const { wrapper } = await mountItem(task(1, 'Buy milk'))

    expect(wrapper.get('[data-slot="checkbox"]').attributes('aria-label')).toBe(
      'Buy milk',
    )
    expect(wrapper.find('[aria-label="Edit task: Buy milk"]').exists()).toBe(
      true,
    )
    expect(wrapper.find('[aria-label="Delete task: Buy milk"]').exists()).toBe(
      true,
    )
  })

  it('saves an edit on Enter', async () => {
    const { wrapper, update } = await mountItem(task(1, 'Buy milk'))

    await wrapper.get('[aria-label="Edit task: Buy milk"]').trigger('click')
    const input = wrapper.get('input')
    expect(inputValue(wrapper)).toBe('Buy milk')
    await input.setValue(' Buy oat milk ')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()

    expect(update).toHaveBeenCalledWith({ title: 'Buy oat milk' })
    expect(wrapper.find('input').exists()).toBe(false)
  })

  it('starts editing on double-click and cancels on Escape', async () => {
    const { wrapper, update } = await mountItem(task(1, 'Buy milk'))

    await wrapper.get('[data-testid="task-title"]').trigger('dblclick')
    await wrapper.get('input').setValue('Something else')
    await wrapper.get('input').trigger('keydown', { key: 'Escape' })
    await flushPromises()

    expect(update).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="task-title"]').text()).toBe('Buy milk')
  })

  it('keeps editing when the save fails or the title is blank', async () => {
    const failing = vi.fn(async () => false)
    const { wrapper } = await mountItem(task(1, 'Buy milk'), {
      update: failing,
    })

    await wrapper.get('[aria-label="Edit task: Buy milk"]').trigger('click')
    await wrapper.get('input').setValue(' ')
    await wrapper.get('input').trigger('keydown', { key: 'Enter' })
    expect(wrapper.get('[role="alert"]').text()).toBe('Title is required')
    expect(failing).not.toHaveBeenCalled()

    await wrapper.get('input').setValue('Buy oat milk')
    await wrapper.get('input').trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(failing).toHaveBeenCalledWith({ title: 'Buy oat milk' })
    expect(inputValue(wrapper)).toBe('Buy oat milk')
  })

  it('deletes through its button', async () => {
    const { wrapper, remove } = await mountItem(task(1, 'Buy milk'))

    await wrapper.get('[aria-label="Delete task: Buy milk"]').trigger('click')

    expect(remove).toHaveBeenCalledOnce()
  })

  it('disables its controls while a change is pending', async () => {
    const { wrapper } = await mountItem(task(1, 'Buy milk'), { pending: true })

    expect(
      wrapper.get('[data-slot="checkbox"]').attributes('disabled'),
    ).toBeDefined()
    expect(
      wrapper.get('[aria-label="Edit task: Buy milk"]').attributes('disabled'),
    ).toBeDefined()
    expect(
      wrapper
        .get('[aria-label="Delete task: Buy milk"]')
        .attributes('disabled'),
    ).toBeDefined()
  })

  it('strikes through a completed task', async () => {
    const { wrapper } = await mountItem(task(1, 'Buy milk', true))

    expect(wrapper.get('[data-testid="task-title"]').classes()).toContain(
      'line-through',
    )
    expect(wrapper.get('li').attributes('data-completed')).toBe('true')
  })
})

describe('TaskList', () => {
  it('emits empty after deleting the last task', async () => {
    const tasks = ref([task(1, 'only')])
    const wrapper = await mountSuspended(TaskList, {
      props: {
        tasks: tasks.value,
        isPending: () => false,
        update: vi.fn(),
        remove: async () => {
          tasks.value = []
          await wrapper.setProps({ tasks: [] })
          return true
        },
      },
    })

    await wrapper.get('[aria-label="Delete task: only"]').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('empty')).toHaveLength(1)
  })
})

describe('TaskFilter', () => {
  it('marks the active filter and renders the list inside its panel', async () => {
    const wrapper = await mountSuspended(TaskFilter, {
      props: { modelValue: 'pending' },
      slots: { default: () => 'the list' },
    })

    const selected = wrapper
      .findAll('[role="tab"]')
      .filter((t) => t.attributes('aria-selected') === 'true')
    expect(selected.map((t) => t.text())).toEqual(['Pending'])
    expect(wrapper.get('[role="tabpanel"]').text()).toBe('the list')
  })

  it('updates the model when another tab is chosen', async () => {
    const wrapper = await mountSuspended(TaskFilter, {
      props: { modelValue: 'all' },
    })

    const completed = wrapper
      .findAll('[role="tab"]')
      .find((t) => t.text() === 'Completed')!
    await completed.trigger('mousedown', { button: 0, ctrlKey: false })

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['completed'])
  })
})
