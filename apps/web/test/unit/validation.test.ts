import { describe, expect, it } from 'vitest'
import {
  codePoints,
  validateEmail,
  validatePassword,
  validateTitle,
} from '../../app/utils/validation'

describe('codePoints', () => {
  it('counts an emoji once', () => {
    expect('😀'.length).toBe(2)
    expect(codePoints('😀')).toBe(1)
  })
})

describe('validateEmail', () => {
  const local = 'a'.repeat(64)
  it.each([
    ['ana@example.com', 'ana@example.com'],
    ['Ana.Silva@Example.COM', 'ana.silva@example.com'],
    ['  ana@example.com\t', 'ana@example.com'],
    [`${local}@${'b'.repeat(185)}.com`, `${local}@${'b'.repeat(185)}.com`],
  ])('accepts %j as %j', (raw, want) => {
    expect(validateEmail(raw)).toEqual({ value: want, problem: null })
  })

  it.each([
    '',
    '   ',
    'Ana <ana@example.com>',
    'ana.example.com',
    'ana@',
    'ana@example.com, bob@example.com',
    `${local}@${'b'.repeat(186)}.com`,
  ])('rejects %j', (raw) => {
    expect(validateEmail(raw).problem).not.toBeNull()
  })
})

describe('validatePassword', () => {
  it.each([
    ['abcdefg', false],
    ['abcdefgh', true],
    ['a'.repeat(64), true],
    ['a'.repeat(65), false],
    ['🔑'.repeat(8), true],
    ['🔑'.repeat(64), true],
    ['🔑'.repeat(65), false],
    ['  abcdef', true],
    ['', false],
  ])('%j valid: %s', (password, valid) => {
    expect(validatePassword(password) === null).toBe(valid)
  })
})

describe('validateTitle', () => {
  it.each([
    ['Buy milk', 'Buy milk'],
    ['\t Buy milk \n', 'Buy milk'],
    ['Buy  milk', 'Buy  milk'],
    ['a'.repeat(200), 'a'.repeat(200)],
    ['😀'.repeat(200), '😀'.repeat(200)],
    [`  ${'a'.repeat(200)}  `, 'a'.repeat(200)],
  ])('accepts %j', (raw, want) => {
    expect(validateTitle(raw)).toEqual({ value: want, problem: null })
  })

  it.each(['', ' \t\n 　', 'a'.repeat(201), '😀'.repeat(201)])(
    'rejects %j',
    (raw) => {
      expect(validateTitle(raw).problem).not.toBeNull()
    },
  )
})
