import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/services/client'
import type { User } from '~/types/api'

const ana: User = {
  id: 1,
  email: 'ana@example.com',
  createdAt: '2026-09-25T12:00:00Z',
}

/** Makes GET /auth/me answer as if `user` were signed in, or nobody. */
function signedIn(user: User | null) {
  const { $api } = useNuxtApp()
  return vi.spyOn($api.auth, 'me').mockImplementation(async () => {
    if (user) return user
    throw new ApiError(401, 'UNAUTHENTICATED', 'Log in to continue')
  })
}

describe('auth middleware and store', () => {
  beforeEach(() => {
    // A fresh page load: nobody checked yet.
    const auth = useAuthStore()
    auth.user = null
    auth.checked = false
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('sends signed-out users to /login', async () => {
    signedIn(null)
    const router = useRouter()

    // The test app starts on /; a query makes this a real navigation.
    await router.push('/?next=1')

    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('lets signed-out users open the guest pages', async () => {
    signedIn(null)
    const router = useRouter()

    await router.push('/register')

    expect(router.currentRoute.value.path).toBe('/register')
  })

  it('sends signed-in users away from the guest pages', async () => {
    signedIn(ana)
    const router = useRouter()

    await router.push('/login')

    expect(router.currentRoute.value.path).toBe('/')
    expect(useAuthStore().user).toEqual(ana)
  })

  it('asks the API only once per page load', async () => {
    const me = signedIn(ana)
    const router = useRouter()

    await router.push('/')
    await router.push('/?status=pending')

    expect(me).toHaveBeenCalledOnce()
  })

  it('logout clears the store and lands on /login', async () => {
    signedIn(ana)
    const { $api } = useNuxtApp()
    const logout = vi.spyOn($api.auth, 'logout').mockResolvedValue(undefined)
    const router = useRouter()
    await router.push('/')

    await useAuthStore().logout()

    expect(logout).toHaveBeenCalledOnce()
    expect(useAuthStore().user).toBeNull()
    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('logout still signs out locally when the session had already ended', async () => {
    signedIn(ana)
    const { $api } = useNuxtApp()
    vi.spyOn($api.auth, 'logout').mockRejectedValue(
      new ApiError(401, 'UNAUTHENTICATED', ''),
    )
    await useRouter().push('/')

    await useAuthStore().logout()

    expect(useAuthStore().user).toBeNull()
    expect(useRouter().currentRoute.value.path).toBe('/login')
  })
})
