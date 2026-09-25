import { ofetch } from 'ofetch'
import { describe, expect, it, vi } from 'vitest'
import { authService } from '../../app/services/auth'
import { ApiError, createClient } from '../../app/services/client'
import { tasksService } from '../../app/services/tasks'

type Reply = {
  status: number
  body?: unknown
  headers?: Record<string, string>
}

/** A client whose fetch answers from `reply` and records every URL. */
function setup(reply: (url: string) => Reply) {
  const urls: string[] = []
  const fetch = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input instanceof Request ? input.url : input)
    urls.push(url)
    const { status, body, headers = {} } = reply(url)
    return new Response(body === undefined ? null : JSON.stringify(body), {
      status,
      headers:
        body === undefined
          ? headers
          : { 'Content-Type': 'application/json', ...headers },
    })
  })
  const onSessionLost = vi.fn()
  const client = createClient(ofetch.create({}, { fetch }), onSessionLost)
  return {
    auth: authService(client),
    tasks: tasksService(client),
    urls,
    onSessionLost,
  }
}

const error = (code: string, extra: object = {}) => ({
  error: { code, message: code, ...extra },
})

async function rejection(promise: Promise<unknown>): Promise<ApiError> {
  try {
    await promise
  } catch (e) {
    return e as ApiError
  }
  throw new Error('expected the call to fail')
}

describe('API client', () => {
  it('unwraps data and builds URLs', async () => {
    const task = {
      id: 1,
      title: 'Buy milk',
      completed: false,
      createdAt: '',
      updatedAt: '',
    }
    const api = setup((url) =>
      url.includes('/tasks/1')
        ? { status: 204 }
        : { status: 200, body: { data: [task] } },
    )

    expect(await api.tasks.list('pending')).toEqual([task])
    expect(await api.tasks.list('all')).toEqual([task])
    expect(await api.tasks.remove(1)).toBeUndefined()
    expect(api.urls).toEqual([
      '/api/v1/tasks?status=pending',
      '/api/v1/tasks',
      '/api/v1/tasks/1',
    ])
  })

  it('maps validation errors with their fields', async () => {
    const api = setup(() => ({
      status: 400,
      body: error('VALIDATION_FAILED', {
        fields: { title: 'Title is required' },
      }),
    }))

    const e = await rejection(api.tasks.create(' '))
    expect(e).toBeInstanceOf(ApiError)
    expect(e).toMatchObject({
      status: 400,
      code: 'VALIDATION_FAILED',
      fields: { title: 'Title is required' },
    })
  })

  it('reads Retry-After on TOO_MANY_ATTEMPTS', async () => {
    const api = setup(() => ({
      status: 429,
      body: error('TOO_MANY_ATTEMPTS'),
      headers: { 'Retry-After': '120' },
    }))

    const e = await rejection(api.auth.login('ana@example.com', 'wrong'))
    expect(e).toMatchObject({
      status: 429,
      code: 'TOO_MANY_ATTEMPTS',
      retryAfter: 120,
    })
  })

  it('reports a network failure as NETWORK_ERROR', async () => {
    const api = setup(() => {
      throw new TypeError('fetch failed')
    })

    const e = await rejection(api.tasks.list('all'))
    expect(e).toMatchObject({ status: 0, code: 'NETWORK_ERROR' })
  })

  it('calls onSessionLost when a session expires mid-use', async () => {
    const api = setup(() => ({ status: 401, body: error('UNAUTHENTICATED') }))

    await rejection(api.tasks.list('all'))
    expect(api.onSessionLost).toHaveBeenCalledOnce()
  })

  it('does not treat a wrong password as a lost session', async () => {
    const api = setup(() => ({
      status: 401,
      body: error('INVALID_CREDENTIALS'),
    }))

    const e = await rejection(api.auth.login('ana@example.com', 'wrong'))
    expect(e.code).toBe('INVALID_CREDENTIALS')
    expect(api.onSessionLost).not.toHaveBeenCalled()
  })

  it('does not treat 401 from /auth/me as a lost session', async () => {
    const api = setup(() => ({ status: 401, body: error('UNAUTHENTICATED') }))

    await rejection(api.auth.me())
    expect(api.onSessionLost).not.toHaveBeenCalled()
  })
})
