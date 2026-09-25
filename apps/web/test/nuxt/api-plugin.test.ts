import { describe, expect, it } from 'vitest'

describe('api plugin', () => {
  it('provides the auth and tasks services', () => {
    const { $api } = useNuxtApp()
    expect(Object.keys($api.auth).sort()).toEqual([
      'login',
      'logout',
      'me',
      'register',
    ])
    expect(Object.keys($api.tasks).sort()).toEqual([
      'create',
      'list',
      'remove',
      'update',
    ])
  })
})
