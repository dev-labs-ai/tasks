import { describe, expect, it } from 'vitest'
import { formatWait } from '../../app/utils/format'

describe('formatWait', () => {
  it.each([
    [0, '1 second'],
    [1, '1 second'],
    [45, '45 seconds'],
    [60, '1 minute'],
    [61, '2 minutes'],
    [900, '15 minutes'],
  ])('%i seconds reads %j', (seconds, want) => {
    expect(formatWait(seconds)).toBe(want)
  })
})
