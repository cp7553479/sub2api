import { defineStore } from 'pinia'
import { ref, computed, readonly } from 'vue'
import { authAPI, type LoginResponse } from '@/api'
import type { User, LoginRequest, RegisterRequest, AuthResponse } from '@/types'

const AUTO_REFRESH_INTERVAL = 60_000
const TOKEN_REFRESH_BUFFER = 120_000

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(null)
  const runMode = ref<'standard' | 'simple'>('standard')
  let refreshInterval: ReturnType<typeof setInterval> | null = null
  let tokenRefreshTimeout: ReturnType<typeof setTimeout> | null = null
  const isAuthenticated = computed(() => Boolean(token.value && user.value))
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isSimpleMode = computed(() => runMode.value === 'simple')

  function stopRefresh(): void {
    if (refreshInterval) clearInterval(refreshInterval)
    if (tokenRefreshTimeout) clearTimeout(tokenRefreshTimeout)
    refreshInterval = null
    tokenRefreshTimeout = null
  }

  function clearAuth(): void {
    stopRefresh()
    token.value = null
    user.value = null
    authAPI.clearAuthToken()
  }

  function scheduleTokenRefresh(): void {
    if (tokenRefreshTimeout) clearTimeout(tokenRefreshTimeout)
    const expiresAt = authAPI.getTokenExpiresAt()
    if (!expiresAt || !authAPI.getRefreshToken()) return
    tokenRefreshTimeout = setTimeout(async () => {
      try {
        const session = await authAPI.refreshToken()
        token.value = session.access_token
        scheduleTokenRefresh()
      } catch (error) {
        console.error('Session refresh failed:', error instanceof Error ? error.message : 'Session refresh rejected')
        const status = (error as { response?: { status?: number } }).response?.status
        if (status && status >= 400 && status < 500 && status !== 429) clearAuth()
      }
    }, Math.max(0, expiresAt - Date.now() - Math.min(TOKEN_REFRESH_BUFFER, (expiresAt - Date.now()) / 2)))
  }

  function startRefresh(): void {
    stopRefresh()
    refreshInterval = setInterval(() => {
      void refreshUser().catch((error) => console.error('User refresh failed:', error))
    }, AUTO_REFRESH_INTERVAL)
    scheduleTokenRefresh()
  }

  function applyUser(data: User & { run_mode?: 'standard' | 'simple' }): User {
    if (data.run_mode) runMode.value = data.run_mode
    const { run_mode: _runMode, ...profile } = data
    user.value = profile
    token.value = authAPI.getAuthToken()
    localStorage.setItem('auth_user', JSON.stringify(profile))
    return profile
  }

  async function refreshUser(): Promise<User> {
    if (!authAPI.getAuthToken()) throw new Error('Not authenticated')
    try {
      const { data } = await authAPI.getCurrentUser()
      return applyUser(data)
    } catch (error) {
      if ([401, 403].includes((error as { status?: number }).status ?? 0)) clearAuth()
      throw error
    }
  }

  async function checkAuth(): Promise<void> {
    if (!authAPI.getAuthToken()) return
    try {
      await refreshUser()
      startRefresh()
    } catch (error) {
      clearAuth()
      throw error
    }
  }

  function setAuthFromResponse(response: AuthResponse): void {
    applyUser(response.user)
    startRefresh()
  }

  async function login(credentials: LoginRequest): Promise<LoginResponse> {
    clearAuth()
    try {
      const response = await authAPI.login(credentials)
      setAuthFromResponse(response)
      return response
    } catch (error) {
      clearAuth()
      throw error
    }
  }

  async function register(credentials: RegisterRequest): Promise<User | null> {
    clearAuth()
    try {
      const response = await authAPI.register(credentials)
      if ('requires_email_confirmation' in response) return null
      setAuthFromResponse(response)
      return user.value
    } catch (error) {
      clearAuth()
      throw error
    }
  }

  async function logout(): Promise<void> {
    await authAPI.logout()
    clearAuth()
  }

  return {
    user, token, runMode: readonly(runMode), isAuthenticated, isAdmin, isSimpleMode,
    login, register, logout, checkAuth, refreshUser
  }
})
