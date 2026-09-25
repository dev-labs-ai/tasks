// https://nuxt.com/docs/api/configuration/nuxt-config
const apiUrl = process.env.API_URL ?? 'http://localhost:8080'

export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },

  // Every screen but login and register needs a session, so the app is
  // client-rendered and ships as static files (tech spec §1).
  ssr: false,

  typescript: {
    strict: true,
  },

  nitro: {
    // Development only: the browser calls /api on the Nuxt origin and
    // Nitro forwards it to the Go API. The mount prefix is stripped before
    // forwarding, so the target carries /api back. The Host header is kept,
    // which the API's cross-origin check compares against Origin.
    devProxy: {
      '/api': { target: `${apiUrl}/api` },
    },
  },
})
