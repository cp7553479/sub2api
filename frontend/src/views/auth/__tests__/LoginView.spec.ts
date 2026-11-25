import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import LoginView from '../LoginView.vue'
import RegisterView from '../RegisterView.vue'

const { auth, app, push } = vi.hoisted(() => ({
  auth: { login: vi.fn(), register: vi.fn(), isAdmin: false },
  app: { siteName: 'Sub2API', showError: vi.fn(), publicSettingsLoaded: true, cachedPublicSettings: {} as Record<string, unknown> },
  push: vi.fn()
}))
vi.mock('@/stores', () => ({ useAuthStore: () => auth, useAppStore: () => app }))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push }), useRoute: () => ({ query: {} }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/layout', () => ({ AuthLayout: { template: '<main><slot /><slot name="footer" /></main>' } }))
const options = { global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } } }

describe('platform sign-in screens', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    app.cachedPublicSettings = {}
    auth.login.mockResolvedValue({})
    auth.register.mockResolvedValue(null)
  })
  it('signs in with existing credentials and routes to business resources', async () => {
    const wrapper = mount(LoginView, options)
    await wrapper.get('#email').setValue('user@example.test')
    await wrapper.get('#password').setValue('secret')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(auth.login).toHaveBeenCalledWith({ email: 'user@example.test', password: 'secret' })
    expect(push).toHaveBeenCalledWith('/dashboard')
    expect(wrapper.find('input[autocomplete="one-time-code"]').exists()).toBe(false)
  })
  it('keeps submission disabled until the platform and business admission finish', async () => {
    let resolveLogin!: () => void
    auth.login.mockImplementationOnce(() => new Promise<void>((resolve) => { resolveLogin = resolve }))
    const wrapper = mount(LoginView, options)
    await wrapper.get('form').trigger('submit')
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
    resolveLogin()
    await flushPromises()
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeUndefined()
  })
  it('retains configured site-document acknowledgement without adding native authentication data', async () => {
    app.cachedPublicSettings = {
      login_agreement_enabled: true, login_agreement_mode: 'checkbox', login_agreement_revision: 'terms-v1',
      login_agreement_documents: [{ id: 'terms', title: 'Terms', content_md: 'Terms content' }]
    }
    const wrapper = mount(LoginView, options)
    await wrapper.get('#email').setValue('user@example.test')
    await wrapper.get('#password').setValue('secret')
    await wrapper.get('form').trigger('submit')
    expect(auth.login).not.toHaveBeenCalled()
    await wrapper.get('#login-agreement-consent').setValue(true)
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(auth.login).toHaveBeenCalledWith({ email: 'user@example.test', password: 'secret' })
    expect(JSON.parse(localStorage.getItem('sub2api_login_agreement_consent')!).revision).toBe('terms-v1')
  })
  it('shows a platform error without navigating as an authenticated user', async () => {
    auth.login.mockRejectedValueOnce(new Error('Account disabled'))
    const wrapper = mount(LoginView, options)
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(app.showError).toHaveBeenCalledWith('Account disabled')
    expect(push).not.toHaveBeenCalled()
  })
  it('shows email confirmation after signup without faking an active session', async () => {
    const wrapper = mount(RegisterView, options)
    await wrapper.get('#email').setValue('new@example.test')
    await wrapper.get('#password').setValue('secret')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(auth.register).toHaveBeenCalledWith({ email: 'new@example.test', password: 'secret' })
    expect(wrapper.get('[role="status"]').text()).toBe('auth.emailConfirmation')
    expect(push).not.toHaveBeenCalled()
  })
})
