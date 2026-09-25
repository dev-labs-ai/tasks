<script setup lang="ts">
import type { TaskStatus } from '~/types/api'

// Tabs that filter the list; the list itself is the active tab's panel.
const status = defineModel<TaskStatus>({ required: true })

const options: { value: TaskStatus; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'pending', label: 'Pending' },
  { value: 'completed', label: 'Completed' },
]
</script>

<template>
  <Tabs
    :model-value="status"
    @update:model-value="(value) => (status = value as TaskStatus)"
  >
    <TabsList aria-label="Filter tasks">
      <TabsTrigger
        v-for="option in options"
        :key="option.value"
        :value="option.value"
      >
        {{ option.label }}
      </TabsTrigger>
    </TabsList>
    <TabsContent :value="status" class="mt-4">
      <slot />
    </TabsContent>
  </Tabs>
</template>
