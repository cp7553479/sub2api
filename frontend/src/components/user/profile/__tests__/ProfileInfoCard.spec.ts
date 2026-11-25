import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ProfileInfoCard from '../ProfileInfoCard.vue'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key })
}))
const user = {
  id: 5, username: 'Alice', email: 'alice@example.test', avatar_url: '/business-avatar.png',
  role: 'user', status: 'active', balance: 17, concurrency: 4, created_at: '2026-04-20T00:00:00Z'
}
const render = () => mount(ProfileInfoCard, {
  props: { user: user as never },
  global: { stubs: { ProfileAvatarCard: true, ProfileEditForm: true } }
})

describe('business profile overview', () => {
  it('retains the real business avatar, email, balance and concurrency', () => {
    const wrapper = render()
    expect(wrapper.get('img').attributes('src')).toBe('/business-avatar.png')
    expect(wrapper.text()).toContain('alice@example.test')
    expect(wrapper.get('[data-testid="profile-overview-metric-balance"]').text()).toContain('$17.00')
    expect(wrapper.get('[data-testid="profile-overview-metric-concurrency"]').text()).toContain('4')
  })

  it('does not expose native identity linking or profile-source controls', () => {
    const wrapper = render()
    expect(wrapper.find('[data-testid="profile-auth-bindings-panel"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('authBindings')
  })
})
