<script setup lang="ts">
import type { TaskPatch } from '~/services/tasks'
import type { Task } from '~/types/api'
import TaskItem from './TaskItem.vue'

const props = defineProps<{
  tasks: readonly Task[]
  isPending: (id: number) => boolean
  update: (id: number, patch: TaskPatch) => Promise<boolean>
  remove: (id: number) => Promise<boolean>
}>()

const emit = defineEmits<{
  /** The last task was deleted; the page moves focus elsewhere. */
  empty: []
}>()

// Keyed by task ID: refs collected in a v-for are not guaranteed to follow
// the array order.
const items = new Map<number, InstanceType<typeof TaskItem>>()

function setItem(id: number, el: unknown) {
  if (el) items.set(id, el as InstanceType<typeof TaskItem>)
  else items.delete(id)
}

// After a delete, focus goes to the task that took its place, or the one
// before it at the end of the list (tech spec §7.5).
async function removeAt(index: number, id: number) {
  if (!(await props.remove(id))) return
  await nextTick()
  const next = props.tasks[Math.min(index, props.tasks.length - 1)]
  if (next) items.get(next.id)?.focus()
  else emit('empty')
}
</script>

<template>
  <ul class="grid gap-2" aria-label="Tasks">
    <TaskItem
      v-for="(task, index) in tasks"
      :key="task.id"
      :ref="(el) => setItem(task.id, el)"
      :task="task"
      :pending="isPending(task.id)"
      :update="(patch) => update(task.id, patch)"
      :remove="() => removeAt(index, task.id)"
    />
  </ul>
</template>
