import type { $Fetch } from 'ofetch'
import { authService } from '~/services/auth'
import { createClient } from '~/services/client'
import { tasksService } from '~/services/tasks'

// One API client for the app, reached as useNuxtApp().$api (tech spec §7.3).
export default defineNuxtPlugin((nuxtApp) => {
  // Nuxt's $fetch is an ofetch instance; its Nitro typing adds typed
  // routes this client does not use.
  const client = createClient($fetch as unknown as $Fetch, async () => {
    await nuxtApp.runWithContext(() => {
      useAuthStore().clear()
      return navigateTo('/login')
    })
  })

  return {
    provide: {
      api: {
        auth: authService(client),
        tasks: tasksService(client),
      },
    },
  }
})
