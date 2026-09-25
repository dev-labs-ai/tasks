<script setup lang="ts">
import { PlusIcon } from '@lucide/vue'
import type { ComponentPublicInstance } from 'vue'
import { toApiError } from '~/services/client'
import type { Task } from '~/types/api'

const props = defineProps<{
  /** Creates the task; rejects with ApiError when the API refuses it. */
  create: (title: string) => Promise<Task>
}>()

const title = ref('')
const error = ref<string | null>(null)
const submitting = ref(false)
const inputId = useId()
const input = useTemplateRef<ComponentPublicInstance>('input')

function focus() {
  ;(input.value?.$el as HTMLInputElement | undefined)?.focus()
}

async function submit() {
  error.value = null
  const checked = validateTitle(title.value)
  if (checked.problem) {
    error.value = checked.problem
    return
  }

  submitting.value = true
  try {
    await props.create(checked.value)
    title.value = ''
  } catch (e) {
    // The text stays in the input so the user can retry (PRD FR-11).
    const apiError = toApiError(e)
    error.value =
      apiError.fields.title ??
      (apiError.code === 'NETWORK_ERROR'
        ? apiError.message
        : 'Could not add the task. Please try again.')
  } finally {
    submitting.value = false
    await nextTick()
    focus()
  }
}

defineExpose({ focus })
</script>

<template>
  <form class="flex items-start gap-2" novalidate @submit.prevent="submit">
    <div class="grid flex-1 gap-1.5">
      <Label :for="inputId" class="sr-only">New task</Label>
      <Input
        :id="inputId"
        ref="input"
        v-model="title"
        placeholder="Add a task…"
        autocomplete="off"
        :aria-invalid="error ? true : undefined"
        :aria-describedby="error ? `${inputId}-error` : undefined"
      />
      <p
        v-if="error"
        :id="`${inputId}-error`"
        role="alert"
        class="text-sm text-destructive"
      >
        {{ error }}
      </p>
    </div>
    <Button type="submit" :disabled="submitting">
      <PlusIcon aria-hidden="true" />
      Add
    </Button>
  </form>
</template>
