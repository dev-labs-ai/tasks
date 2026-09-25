// Sends signed-out users to /login and signed-in users away from the
// guest pages. This is only for the user's convenience: the API enforces
// access on every request (tech spec §7.3).
export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuthStore()
  const user = await auth.fetchUser()
  const guestPage = to.meta.guest === true

  if (!user && !guestPage) return navigateTo('/login')
  if (user && guestPage) return navigateTo('/')
})
