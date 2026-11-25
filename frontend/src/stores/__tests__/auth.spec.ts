import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'

const { login, register, logout, getCurrentUser, refreshToken } = vi.hoisted(() => ({
  login: vi.fn(), register: vi.fn(), logout: vi.fn(), getCurrentUser: vi.fn(), refreshToken: vi.fn()
}))
vi.mock('@/api', () => ({ authAPI: {
  login, register, logout, getCurrentUser, refreshToken,
  getAuthToken: () => localStorage.getItem('auth_token'),
  getRefreshToken: () => localStorage.getItem('refresh_token'),
  getTokenExpiresAt: () => Number(localStorage.getItem('token_expires_at')) || null,
  clearAuthToken: () => localStorage.clear()
} }))
const user = { id: 7, role: 'user', email: 'user@example.test', balance: 3 }

describe('Supabase-backed auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
    vi.clearAllMocks()
    localStorage.clear()
    login.mockImplementation(async () => {
      localStorage.setItem('auth_token', 'platform-access')
      localStorage.setItem('refresh_token', 'platform-refresh')
      localStorage.setItem('token_expires_at', String(Date.now() + 3600_000))
      return { access_token: 'platform-access', refresh_token: 'platform-refresh', expires_in: 3600, user }
    })
  })
  afterEach(() => { vi.clearAllTimers(); vi.useRealTimers() })

  it('stores the business identity and platform access token on successful admission', async () => {
    const store = useAuthStore()
    await store.login({ email: 'user@example.test', password: 'secret' })
    expect(store.user).toEqual(user)
    expect(store.token).toBe('platform-access')
    expect(store.isAuthenticated).toBe(true)
    expect(store.isAdmin).toBe(false)
  })

  it('restores current permissions from business me instead of trusting cached admin metadata', async () => {
    localStorage.setItem('auth_token', 'platform-access')
    localStorage.setItem('auth_user', JSON.stringify({ ...user, role: 'admin' }))
    getCurrentUser.mockResolvedValueOnce({ data: user })
    const store = useAuthStore()
    expect(store.isAuthenticated).toBe(false)
    await store.checkAuth()
    expect(store.isAdmin).toBe(false)
    expect(store.user).toEqual(user)
  })

  it('clears invalid or disabled sessions during restoration', async () => {
    localStorage.setItem('auth_token', 'platform-access')
    getCurrentUser.mockRejectedValueOnce({ status: 403 })
    const store = useAuthStore()
    await expect(store.checkAuth()).rejects.toMatchObject({ status: 403 })
    expect(store.isAuthenticated).toBe(false)
    expect(localStorage.length).toBe(0)
  })

  it('does not invent a session for signup pending email confirmation', async () => {
    register.mockResolvedValueOnce({ requires_email_confirmation: true })
    const store = useAuthStore()
    await expect(store.register({ email: 'user@example.test', password: 'secret' })).resolves.toBeNull()
    expect(store.isAuthenticated).toBe(false)
  })

  it('refreshes short-lived platform tokens without entering an immediate refresh loop', async () => {
    login.mockImplementationOnce(async () => {
      localStorage.setItem('auth_token', 'short-access')
      localStorage.setItem('refresh_token', 'short-refresh')
      localStorage.setItem('token_expires_at', String(Date.now() + 1000))
      return { access_token: 'short-access', refresh_token: 'short-refresh', expires_in: 1, user }
    })
    refreshToken.mockImplementation(async () => {
      localStorage.setItem('auth_token', 'rotated-access')
      localStorage.setItem('token_expires_at', String(Date.now() + 60_000))
      return { access_token: 'rotated-access', refresh_token: 'rotated-refresh', expires_in: 60 }
    })
    const store = useAuthStore()
    await store.login({ email: 'user@example.test', password: 'secret' })
    await vi.advanceTimersByTimeAsync(400)
    expect(refreshToken).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(1000)
    expect(refreshToken).toHaveBeenCalledTimes(1)
    expect(store.token).toBe('rotated-access')
  })

  it('does not report successful logout if the platform cannot revoke the session', async () => {
    const store = useAuthStore()
    await store.login({ email: 'user@example.test', password: 'secret' })
    logout.mockRejectedValueOnce(new Error('Auth unavailable'))
    await expect(store.logout()).rejects.toThrow('Auth unavailable')
    expect(store.isAuthenticated).toBe(true)
    logout.mockResolvedValueOnce(undefined)
    await store.logout()
    expect(store.isAuthenticated).toBe(false)
    expect(localStorage.length).toBe(0)
  })
})
