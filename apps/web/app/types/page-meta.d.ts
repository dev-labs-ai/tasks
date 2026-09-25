declare module '#app' {
  interface PageMeta {
    /** Only for signed-out users: login and register. */
    guest?: boolean
  }
}

export {}
