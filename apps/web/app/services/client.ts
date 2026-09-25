import type { $Fetch, FetchError, FetchOptions } from 'ofetch'
import type { DataBody, ErrorBody, ErrorCode } from '~/types/api'

/** An API failure, or NETWORK_ERROR when no response arrived. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: ErrorCode | 'NETWORK_ERROR',
    message: string,
    /** Per-field problems from VALIDATION_FAILED. */
    readonly fields: Record<string, string> = {},
    /** Seconds from Retry-After, sent with TOO_MANY_ATTEMPTS. */
    readonly retryAfter: number | null = null,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) return error
  const fetchError = error as Partial<FetchError<Partial<ErrorBody>>>
  const response = fetchError.response
  if (!response) {
    return new ApiError(
      0,
      'NETWORK_ERROR',
      'Could not reach the server. Check your connection.',
    )
  }
  const body = fetchError.data?.error
  const retryAfter = Number.parseInt(
    response.headers.get('Retry-After') ?? '',
    10,
  )
  return new ApiError(
    response.status,
    body?.code ?? 'INTERNAL_ERROR',
    body?.message ?? 'Something went wrong',
    body?.fields ?? {},
    Number.isNaN(retryAfter) ? null : retryAfter,
  )
}

/**
 * True when a response means the session is gone, for example expired.
 * GET /auth/me answers 401 whenever nobody is logged in, which is how the
 * app finds that out, so it does not count.
 */
export function isSessionLost(
  path: string,
  status: number,
  body: unknown,
): boolean {
  const code = (body as Partial<ErrorBody> | undefined)?.error?.code
  return (
    status === 401 && code === 'UNAUTHENTICATED' && !path.endsWith('/auth/me')
  )
}

/**
 * Builds the API client on top of a $fetch instance: base URL /api/v1 and
 * a callback for lost sessions. The login form's 401 INVALID_CREDENTIALS
 * does not trigger it.
 */
export function createClient(
  base: $Fetch,
  onSessionLost: () => void | Promise<void>,
): $Fetch {
  return base.create({
    baseURL: '/api/v1',
    async onResponseError({ request, response }) {
      const path = typeof request === 'string' ? request : request.url
      if (isSessionLost(path, response.status, response._data)) {
        await onSessionLost()
      }
    },
  })
}

/** Sends a request and unwraps { data }, turning failures into ApiError. */
export async function call<T>(
  client: $Fetch,
  path: string,
  options: FetchOptions<'json'> = {},
): Promise<T> {
  try {
    const body = await client<DataBody<T> | undefined>(path, options)
    return body?.data as T
  } catch (error) {
    throw toApiError(error)
  }
}
