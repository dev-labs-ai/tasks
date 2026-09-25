import { defineStore } from 'pinia'
import { ApiError } from '~/services/client'
import type { User } from '~/types/api'

// The signed-in user, shared by the route middleware and the layout.
export const useAuthStore = defineStore('auth', () => {
  const { $api } = useNuxtApp()

  const user = ref<User | null>(null)
  /** Whether GET /auth/me has answered since the page loaded. */
  const checked = ref(false)

  /** Asks the API who is signed in, once per page load. */
  async function fetchUser(): Promise<User | null> {
    if (checked.value) return user.value
    try {
      user.value = await $api.auth.me()
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) throw error
      user.value = null
    }
    checked.value = true
    return user.value
  }

  function setUser(value: User) {
    user.value = value
    checked.value = true
  }

  /** Forgets the user locally, after logout or when the session is lost. */
  function clear() {
    user.value = null
    checked.value = true
  }

  async function logout() {
    try {
      await $api.auth.logout()
    } catch (error) {
      // 401: the session had already ended, which is what we wanted.
      if (!(error instanceof ApiError && error.status === 401)) throw error
    }
    clear()
    await navigateTo('/login')
  }

  return { user, checked, fetchUser, setUser, clear, logout }
})
