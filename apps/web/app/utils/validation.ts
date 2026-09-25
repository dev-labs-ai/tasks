// Client-side copies of the API's rules (tech spec §5.6), for instant
// feedback. The API stays the authority. Lengths count Unicode code points,
// like the API and PostgreSQL: `[...s].length`, never `s.length`, which
// counts UTF-16 units and would count an emoji twice.

export const MAX_EMAIL_LENGTH = 254
export const MIN_PASSWORD_LENGTH = 8
export const MAX_PASSWORD_LENGTH = 64
export const MAX_TITLE_LENGTH = 200

export interface Checked {
  /** The normalized value; empty when there is a problem. */
  value: string
  /** What to tell the user, or null when the value is valid. */
  problem: string | null
}

export function codePoints(s: string): number {
  return [...s].length
}

// A bare address: something@something, no spaces. The API parses it fully;
// this catches typos and pasted "Name <address>" forms early.
const EMAIL = /^[^\s@]+@[^\s@]+$/

export function validateEmail(raw: string): Checked {
  const value = raw.trim().toLowerCase()
  if (value === '') return { value: '', problem: 'Email is required' }
  if (codePoints(value) > MAX_EMAIL_LENGTH) {
    return {
      value: '',
      problem: `Email must have at most ${MAX_EMAIL_LENGTH} characters`,
    }
  }
  if (!EMAIL.test(value))
    return { value: '', problem: 'Enter a valid email address' }
  return { value, problem: null }
}

/** Passwords are never trimmed: spaces are part of the password. */
export function validatePassword(password: string): string | null {
  const n = codePoints(password)
  if (n < MIN_PASSWORD_LENGTH || n > MAX_PASSWORD_LENGTH) {
    return `Password must have ${MIN_PASSWORD_LENGTH} to ${MAX_PASSWORD_LENGTH} characters`
  }
  return null
}

export function validateTitle(raw: string): Checked {
  const value = raw.trim()
  const n = codePoints(value)
  if (n === 0) return { value: '', problem: 'Title is required' }
  if (n > MAX_TITLE_LENGTH) {
    return {
      value: '',
      problem: `Title must have at most ${MAX_TITLE_LENGTH} characters`,
    }
  }
  return { value, problem: null }
}
