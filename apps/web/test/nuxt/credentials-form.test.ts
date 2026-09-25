import { mountSuspended } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import CredentialsForm from '~/components/auth/CredentialsForm.vue'
import { ApiError } from '~/services/client'
import type { User } from '~/types/api'

const ana: User = {
  id: 1,
  email: 'ana@example.com',
  createdAt: '2026-09-25T12:00:00Z',
}

async function fill(
  mode: 'login' | 'register',
  email: string,
  password: string,
) {
  const wrapper = await mountSuspended(CredentialsForm, { props: { mode } })
  await wrapper.find('input[type="email"]').setValue(email)
  await wrapper.find('input[type="password"]').setValue(password)
  await wrapper.find('form').trigger('submit')
  await flushPromises()
  return wrapper
}

describe('CredentialsForm', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows client-side errors without calling the API', async () => {
    const register = vi.spyOn(useNuxtApp().$api.auth, 'register')

    const wrapper = await fill('register', 'Ana <ana@example.com>', 'short')

    expect(register).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Enter a valid email address')
    expect(wrapper.text()).toContain('Password must have 8 to 64 characters')
    expect(wrapper.find('input[type="email"]').attributes('aria-invalid')).toBe(
      'true',
    )
  })

  it('shows field errors returned by the API', async () => {
    vi.spyOn(useNuxtApp().$api.auth, 'register').mockRejectedValue(
      new ApiError(400, 'VALIDATION_FAILED', 'The request has invalid fields', {
        email: 'Email must have at most 254 characters',
      }),
    )

    const wrapper = await fill('register', 'ana@example.com', 'correct horse')

    expect(wrapper.text()).toContain('Email must have at most 254 characters')
  })

  it('reports a registered email on the email field', async () => {
    vi.spyOn(useNuxtApp().$api.auth, 'register').mockRejectedValue(
      new ApiError(409, 'EMAIL_TAKEN', 'This email is already registered'),
    )

    const wrapper = await fill('register', 'ana@example.com', 'correct horse')

    expect(wrapper.text()).toContain('This email is already registered')
  })

  it('shows one generic message for a failed login and keeps the input', async () => {
    vi.spyOn(useNuxtApp().$api.auth, 'login').mockRejectedValue(
      new ApiError(401, 'INVALID_CREDENTIALS', 'Invalid email or password'),
    )

    const wrapper = await fill('login', 'ana@example.com', 'wrong password')

    expect(wrapper.get('[data-testid="form-error"]').text()).toBe(
      'Invalid email or password',
    )
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(
      (wrapper.find('input[type="email"]').element as HTMLInputElement).value,
    ).toBe('ana@example.com')
    expect(
      (wrapper.find('input[type="password"]').element as HTMLInputElement)
        .value,
    ).toBe('wrong password')
  })

  it('tells the user how long to wait after too many attempts', async () => {
    vi.spyOn(useNuxtApp().$api.auth, 'login').mockRejectedValue(
      new ApiError(
        429,
        'TOO_MANY_ATTEMPTS',
        'Too many login attempts',
        {},
        120,
      ),
    )

    const wrapper = await fill('login', 'ana@example.com', 'correct horse')

    expect(wrapper.get('[data-testid="form-error"]').text()).toBe(
      'Too many login attempts. Try again in 2 minutes.',
    )
  })

  it('logs in with the normalized email and lands on /', async () => {
    const login = vi
      .spyOn(useNuxtApp().$api.auth, 'login')
      .mockResolvedValue(ana)

    await fill('login', '  Ana@Example.com ', 'correct horse')

    expect(login).toHaveBeenCalledWith('ana@example.com', 'correct horse')
    expect(useAuthStore().user).toEqual(ana)
    expect(useRouter().currentRoute.value.path).toBe('/')
  })
})
