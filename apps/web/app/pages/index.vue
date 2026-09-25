<script setup lang="ts">
import TaskFilter from '~/components/tasks/TaskFilter.vue'
import TaskForm from '~/components/tasks/TaskForm.vue'
import TaskList from '~/components/tasks/TaskList.vue'
import type { TaskStatus } from '~/types/api'

useHead({ title: 'My tasks · Tasks' })

const status = useTaskFilter()
const {
  tasks,
  loading,
  loadError,
  actionError,
  load,
  create,
  update,
  remove,
  isPending,
} = useTasks(status)
const form = useTemplateRef<InstanceType<typeof TaskForm>>('form')

// Deleting the last task replaces the list with the empty state; focus
// goes to the input instead of being lost (tech spec §7.5).
async function removeTask(id: number): Promise<boolean> {
  const removed = await remove(id)
  if (removed && tasks.value.length === 0) {
    await nextTick()
    form.value?.focus()
  }
  return removed
}

const emptyMessages: Record<TaskStatus, string> = {
  all: 'No tasks yet. Add your first one above.',
  pending: 'Nothing pending. Nice work.',
  completed: 'No completed tasks yet.',
}

const actionMessage = computed(() => {
  const error = actionError.value
  if (!error) return null
  if (error.code === 'TASK_NOT_FOUND') return 'That task no longer exists.'
  if (error.code === 'NETWORK_ERROR') return error.message
  if (error.code === 'VALIDATION_FAILED')
    return Object.values(error.fields)[0] ?? error.message
  return 'Could not save the change. Please try again.'
})
</script>

<template>
  <div class="grid gap-6">
    <h1 class="text-2xl font-semibold">My tasks</h1>

    <TaskForm ref="form" :create="create" />

    <TaskFilter v-model="status">
      <div class="grid gap-3">
        <Alert
          v-if="actionMessage"
          variant="destructive"
          data-testid="action-error"
        >
          <AlertDescription>{{ actionMessage }}</AlertDescription>
        </Alert>

        <div
          v-if="loading"
          class="grid gap-2"
          aria-busy="true"
          aria-label="Loading tasks"
        >
          <Skeleton v-for="n in 3" :key="n" class="h-12 w-full rounded-lg" />
        </div>

        <Alert
          v-else-if="loadError"
          variant="destructive"
          data-testid="load-error"
        >
          <AlertDescription
            class="flex flex-wrap items-center justify-between gap-2"
          >
            <span>{{
              loadError.code === 'NETWORK_ERROR'
                ? loadError.message
                : 'Could not load your tasks.'
            }}</span>
            <Button size="sm" variant="outline" @click="load">Try again</Button>
          </AlertDescription>
        </Alert>

        <p
          v-else-if="tasks.length === 0"
          class="rounded-lg border border-dashed px-4 py-8 text-center text-sm text-muted-foreground"
          data-testid="empty"
        >
          {{ emptyMessages[status] }}
        </p>

        <TaskList
          v-else
          :tasks="tasks"
          :is-pending="isPending"
          :update="update"
          :remove="removeTask"
        />
      </div>
    </TaskFilter>
  </div>
</template>
