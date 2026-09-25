import type { $Fetch } from 'ofetch'
import type { User } from '~/types/api'
import { call } from './client'

export function authService(client: $Fetch) {
  return {
    register: (email: string, password: string) =>
      call<User>(client, '/auth/register', {
        method: 'POST',
        body: { email, password },
      }),
    login: (email: string, password: string) =>
      call<User>(client, '/auth/login', {
        method: 'POST',
        body: { email, password },
      }),
    logout: () => call<undefined>(client, '/auth/logout', { method: 'POST' }),
    me: () => call<User>(client, '/auth/me'),
  }
}

export type AuthService = ReturnType<typeof authService>
