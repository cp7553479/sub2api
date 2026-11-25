import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import axios from 'axios'

const { getBusinessUser } = vi.hoisted(() => ({ getBusinessUser: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { get: getBusinessUser } }))
vi.mock('axios', () => ({ default: { request: vi.fn(), post: vi.fn(), isAxiosError: vi.fn(() => false) } }))

const platformRequest = vi.mocked(axios.request)
const platformSession = {
  access_token: 'platform-access', refresh_token: 'platform-refresh', expires_in: 3600,
  token_type: 'bearer', user: { id: 'd64a2c47-6142-4bc7-b61c-927f4ae18e32', email: 'user@example.test' }
}
const businessUser = { id: 17, email: 'user@example.test', role: 'user', balance: 4, run_mode: 'standard' }

describe('platform account sessions', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.stubEnv('VITE_PLATFORM_API_URL', 'https://identity.example.test/')
    vi.clearAllMocks()
    getBusinessUser.mockResolvedValue({ data: businessUser })
  })
  afterEach(() => vi.unstubAllEnvs())

  it('uses the platform session and fetches the separate bigint business identity before committing', async () => {
    platformRequest.mockResolvedValueOnce({ data: platformSession })
    getBusinessUser.mockImplementationOnce(async () => {
      expect(localStorage.getItem('auth_token')).toBeNull()
      return { data: businessUser }
    })
    const { login } = await import('@/api/auth')
    const result = await login({ email: 'user@example.test', password: 'secret' })
    expect(platformRequest).toHaveBeenCalledWith(expect.objectContaining({
      url: 'https://identity.example.test/auth/login', method: 'POST',
      data: { email: 'user@example.test', password: 'secret' }
    }))
    expect(getBusinessUser).toHaveBeenCalledWith('/auth/me', {
      headers: { Authorization: 'Bearer platform-access' }
    })
    expect(result.user.id).toBe(17)
    expect(localStorage.getItem('refresh_token')).toBe('platform-refresh')
    expect(JSON.parse(localStorage.getItem('auth_user')!).id).toBe(17)
  })

  it.each(['login', 'register'] as const)('revokes only the new session after rejected %s admission', async (action) => {
    platformRequest.mockResolvedValueOnce({ data: platformSession }).mockResolvedValueOnce({ data: undefined })
    const rejection = { status: 403, message: 'Account disabled' }
    getBusinessUser.mockRejectedValueOnce(rejection)
    const auth = await import('@/api/auth')
    await expect(auth[action]({ email: 'user@example.test', password: 'secret' })).rejects.toBe(rejection)
    expect(platformRequest).toHaveBeenLastCalledWith(expect.objectContaining({
      url: 'https://identity.example.test/auth/logout', method: 'POST', params: { scope: 'local' },
      headers: expect.objectContaining({ Authorization: 'Bearer platform-access' })
    }))
    expect(localStorage.length).toBe(0)
  })

  it('awaits admission rollback before returning the admission error', async () => {
    let completeRollback!: () => void
    platformRequest.mockResolvedValueOnce({ data: platformSession }).mockImplementationOnce(() => new Promise((resolve) => {
      completeRollback = () => resolve({ data: undefined })
    }))
    const rejection = { status: 503, message: 'Service unavailable' }
    getBusinessUser.mockRejectedValueOnce(rejection)
    const { login } = await import('@/api/auth')
    const rejected = vi.fn()
    const pending = login({ email: 'user@example.test', password: 'secret' }).catch(rejected)
    await vi.waitFor(() => expect(platformRequest).toHaveBeenCalledTimes(2))
    expect(rejected).not.toHaveBeenCalled()
    expect(localStorage.length).toBe(0)
    completeRollback()
    await pending
    expect(rejected).toHaveBeenCalledWith(rejection)
  })

  it('reports failed rollback without logging credentials or hiding the admission error', async () => {
    const diagnostic = vi.spyOn(console, 'error').mockImplementation(() => {})
    platformRequest.mockResolvedValueOnce({ data: platformSession }).mockRejectedValueOnce({
      response: { status: 503 }, config: { headers: { Authorization: 'Bearer platform-access' } }
    })
    const rejection = { status: 403, message: 'Account disabled' }
    getBusinessUser.mockRejectedValueOnce(rejection)
    const { login } = await import('@/api/auth')
    await expect(login({ email: 'user@example.test', password: 'secret' })).rejects.toBe(rejection)
    expect(diagnostic).toHaveBeenCalledWith('Sign-in admission rollback failed', {
      admissionStatus: 403, rollbackStatus: 503
    })
    expect(JSON.stringify(diagnostic.mock.calls)).not.toContain('platform-access')
    expect(localStorage.length).toBe(0)
    diagnostic.mockRestore()
  })

  it('handles a registration requiring email confirmation without issuing a local session', async () => {
    platformRequest.mockResolvedValueOnce({ data: platformSession.user })
    const { register } = await import('@/api/auth')
    await expect(register({ email: 'user@example.test', password: 'secret' }))
      .resolves.toEqual({ requires_email_confirmation: true })
    expect(getBusinessUser).not.toHaveBeenCalled()
    expect(localStorage.length).toBe(0)
  })

  it('awaits actual platform revocation and retains the session when logout fails', async () => {
    localStorage.setItem('auth_token', 'platform-access')
    localStorage.setItem('refresh_token', 'platform-refresh')
    platformRequest.mockRejectedValueOnce(new Error('Auth unavailable'))
    const { logout } = await import('@/api/auth')
    await expect(logout()).rejects.toThrow('Auth unavailable')
    expect(localStorage.getItem('auth_token')).toBe('platform-access')
    expect(platformRequest).toHaveBeenCalledWith(expect.objectContaining({
      url: 'https://identity.example.test/auth/logout', method: 'POST',
      headers: expect.objectContaining({ Authorization: 'Bearer platform-access' })
    }))
    platformRequest.mockResolvedValueOnce({ data: undefined })
    await logout()
    expect(localStorage.length).toBe(0)
  })

  it('updates account credentials only at the platform endpoint', async () => {
    localStorage.setItem('auth_token', 'platform-access')
    platformRequest.mockResolvedValueOnce({ data: platformSession.user })
    const { updateAccount } = await import('@/api/auth')
    await updateAccount({ password: 'new-secret' })
    expect(platformRequest).toHaveBeenCalledWith(expect.objectContaining({
      url: 'https://identity.example.test/auth/update', method: 'POST', data: { password: 'new-secret' }
    }))
    expect(getBusinessUser).not.toHaveBeenCalled()
  })

  it('shares one refresh for concurrent expired-session account updates', async () => {
    localStorage.setItem('auth_token', 'expired-access')
    localStorage.setItem('refresh_token', 'old-refresh')
    localStorage.setItem('token_expires_at', String(Date.now() - 1000))
    vi.mocked(axios.post).mockResolvedValueOnce({ data: platformSession })
    platformRequest.mockResolvedValue({ data: platformSession.user })
    const { updateAccount } = await import('@/api/auth')
    await Promise.all([updateAccount({ email: 'new@example.test' }), updateAccount({ password: 'new-secret' })])
    expect(axios.post).toHaveBeenCalledTimes(1)
    expect(axios.post).toHaveBeenCalledWith('https://identity.example.test/auth/refresh',
      { refresh_token: 'old-refresh' }, expect.any(Object))
    expect(platformRequest).toHaveBeenCalledTimes(2)
    for (const [request] of platformRequest.mock.calls) {
      expect(request).toMatchObject({ headers: { Authorization: 'Bearer platform-access' } })
    }
  })

  it('reports missing platform configuration instead of returning to native login', async () => {
    vi.stubEnv('VITE_PLATFORM_API_URL', '')
    const { login } = await import('@/api/auth')
    await expect(login({ email: 'user@example.test', password: 'secret' })).rejects.toThrow('VITE_PLATFORM_API_URL')
    expect(platformRequest).not.toHaveBeenCalled()
  })
})
