<script setup lang="ts">
import { CheckIcon, PencilIcon, Trash2Icon, XIcon } from '@lucide/vue'
import type { ComponentPublicInstance } from 'vue'
import type { TaskPatch } from '~/services/tasks'
import type { Task } from '~/types/api'

const props = defineProps<{
  task: Task
  /** True while a change to this task is in flight. */
  pending: boolean
  update: (patch: TaskPatch) => Promise<boolean>
  remove: () => Promise<unknown>
}>()

const editing = ref(false)
const draft = ref('')
const editError = ref<string | null>(null)
const editInputId = useId()
const checkbox = useTemplateRef<ComponentPublicInstance>('checkbox')
const editInput = useTemplateRef<ComponentPublicInstance>('editInput')
const editButton = useTemplateRef<ComponentPublicInstance>('editButton')

function focusOf(target: ComponentPublicInstance | null) {
  ;(target?.$el as HTMLElement | undefined)?.focus()
}

async function startEdit() {
  if (props.pending) return
  draft.value = props.task.title
  editError.value = null
  editing.value = true
  await nextTick()
  const el = editInput.value?.$el as HTMLInputElement | undefined
  el?.focus()
  el?.select()
}

async function cancel() {
  editing.value = false
  editError.value = null
  await nextTick()
  focusOf(editButton.value)
}

async function save() {
  const checked = validateTitle(draft.value)
  if (checked.problem) {
    editError.value = checked.problem
    return
  }
  if (checked.value === props.task.title) return cancel()

  // On failure the input stays open with the text; the page shows why.
  if (await props.update({ title: checked.value })) await cancel()
}

function toggle(checked: boolean | 'indeterminate') {
  props.update({ completed: checked === true })
}

defineExpose({ focus: () => focusOf(checkbox.value) })
</script>

<template>
  <li
    class="rounded-lg border bg-background px-3 py-2"
    :data-completed="task.completed || undefined"
  >
    <div class="flex items-center gap-3">
      <Checkbox
        ref="checkbox"
        :model-value="task.completed"
        :disabled="pending"
        :aria-label="task.title"
        @update:model-value="toggle"
      />

      <template v-if="editing">
        <Label :for="editInputId" class="sr-only">Title</Label>
        <Input
          :id="editInputId"
          ref="editInput"
          v-model="draft"
          class="flex-1"
          :disabled="pending"
          :aria-invalid="editError ? true : undefined"
          :aria-describedby="editError ? `${editInputId}-error` : undefined"
          @keydown.enter.prevent="save"
          @keydown.esc.prevent="cancel"
        />
        <Button
          size="icon-sm"
          variant="ghost"
          :disabled="pending"
          aria-label="Save title"
          @click="save"
        >
          <CheckIcon aria-hidden="true" />
        </Button>
        <Button
          size="icon-sm"
          variant="ghost"
          :disabled="pending"
          aria-label="Cancel editing"
          @click="cancel"
        >
          <XIcon aria-hidden="true" />
        </Button>
      </template>
      <template v-else>
        <span
          class="min-w-0 flex-1 break-words"
          :class="{ 'text-muted-foreground line-through': task.completed }"
          data-testid="task-title"
          @dblclick="startEdit"
        >
          {{ task.title }}
        </span>
        <Button
          ref="editButton"
          size="icon-sm"
          variant="ghost"
          :disabled="pending"
          :aria-label="`Edit task: ${task.title}`"
          @click="startEdit"
        >
          <PencilIcon aria-hidden="true" />
        </Button>
        <Button
          size="icon-sm"
          variant="ghost"
          :disabled="pending"
          :aria-label="`Delete task: ${task.title}`"
          @click="remove"
        >
          <Trash2Icon aria-hidden="true" />
        </Button>
      </template>
    </div>
    <p
      v-if="editing && editError"
      :id="`${editInputId}-error`"
      role="alert"
      class="mt-1 text-sm text-destructive"
    >
      {{ editError }}
    </p>
  </li>
</template>
