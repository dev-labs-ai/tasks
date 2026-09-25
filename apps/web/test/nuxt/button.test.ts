import { mountSuspended } from '@nuxt/test-utils/runtime'
import { describe, expect, it } from 'vitest'
import { Button } from '~/components/ui/button'

describe('Button', () => {
  it('renders its slot inside the Nuxt environment', async () => {
    const wrapper = await mountSuspended(Button, {
      slots: { default: () => 'Save' },
    })
    expect(wrapper.text()).toBe('Save')
    expect(wrapper.attributes('data-slot')).toBe('button')
  })
})
