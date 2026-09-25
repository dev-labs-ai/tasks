<script setup lang="ts">
const auth = useAuthStore()
const loggingOut = ref(false)

async function logout() {
  loggingOut.value = true
  try {
    await auth.logout()
  } finally {
    loggingOut.value = false
  }
}
</script>

<template>
  <div class="min-h-dvh bg-muted/40">
    <header class="border-b bg-background">
      <div
        class="mx-auto flex max-w-2xl items-center justify-between gap-4 px-4 py-3"
      >
        <span class="font-semibold">Tasks</span>
        <div class="flex min-w-0 items-center gap-3">
          <span
            class="truncate text-sm text-muted-foreground"
            data-testid="user-email"
          >
            {{ auth.user?.email }}
          </span>
          <Button
            variant="outline"
            size="sm"
            :disabled="loggingOut"
            @click="logout"
          >
            Log out
          </Button>
        </div>
      </div>
    </header>
    <main class="mx-auto max-w-2xl px-4 py-6">
      <slot />
    </main>
  </div>
</template>
