import { apiClient } from './client'
import { refreshAuthTokens, type RefreshTokenResponse } from './tokenRefresh'
import { platformRequest, type PlatformSession, type PlatformUser } from './platform'
import type { LoginRequest, RegisterRequest, AuthResponse, CurrentUserResponse, PublicSettings } from '@/types'

export type { RefreshTokenResponse } from './tokenRefresh'
export type LoginResponse = AuthResponse
export type RegistrationResponse = AuthResponse | { requires_email_confirmation: true }

export function setAuthToken(token: string): void {
  localStorage.setItem('auth_token', token)
}
export function setRefreshToken(token: string): void {
  localStorage.setItem('refresh_token', token)
}
export function setTokenExpiresAt(expiresIn: number): void {
  localStorage.setItem('token_expires_at', String(Date.now() + expiresIn * 1000))
}
export function getAuthToken(): string | null {
  return localStorage.getItem('auth_token')
}
export function getRefreshToken(): string | null {
  return localStorage.getItem('refresh_token')
}
export function getTokenExpiresAt(): number | null {
  const value = localStorage.getItem('token_expires_at')
  return value ? Number(value) : null
}
export function clearAuthToken(): void {
  for (const key of ['auth_token', 'refresh_token', 'auth_user', 'token_expires_at']) {
    localStorage.removeItem(key)
  }
}

async function establishSession(session: PlatformSession): Promise<AuthResponse> {
  let user: CurrentUserResponse
  try {
    const response = await apiClient.get<CurrentUserResponse>('/auth/me', {
      headers: { Authorization: `Bearer ${session.access_token}` }
    })
    user = response.data
  } catch (admissionError) {
    try {
      await platformRequest<void>('/logout', {
        method: 'POST',
        params: { scope: 'local' },
        headers: { Authorization: `Bearer ${session.access_token}` }
      })
    } catch (rollbackError) {
      const status = (error: unknown) => {
        const value = error as { status?: number; response?: { status?: number } }
        return value?.response?.status ?? value?.status
      }
      console.error('Sign-in admission rollback failed', {
        admissionStatus: status(admissionError), rollbackStatus: status(rollbackError)
      })
    }
    throw admissionError
  }
  localStorage.setItem('auth_user', JSON.stringify(user))
  setAuthToken(session.access_token)
  setTokenExpiresAt(session.expires_in)
  setRefreshToken(session.refresh_token)
  return { ...session, user }
}

export async function login({ email, password }: LoginRequest): Promise<LoginResponse> {
  clearAuthToken()
  const session = await platformRequest<PlatformSession>('/login', {
    method: 'POST', data: { email, password }
  })
  return establishSession(session)
}

export async function register({ email, password }: RegisterRequest): Promise<RegistrationResponse> {
  clearAuthToken()
  const result = await platformRequest<PlatformSession | PlatformUser>('/signup', {
    method: 'POST', data: { email, password }
  })
  if (!('access_token' in result) || !result.access_token) {
    return { requires_email_confirmation: true }
  }
  return establishSession(result)
}

export function getCurrentUser() {
  return apiClient.get<CurrentUserResponse>('/auth/me')
}

async function getActiveAuthToken(): Promise<string | null> {
  let token = getAuthToken()
  const expiresAt = getTokenExpiresAt()
  if (token && expiresAt !== null && expiresAt <= Date.now()) {
    token = (await refreshAuthTokens()).access_token
  }
  return token
}

export async function logout(): Promise<void> {
  const token = await getActiveAuthToken()
  if (token) {
    await platformRequest<void>('/logout', {
      method: 'POST', headers: { Authorization: `Bearer ${token}` }
    })
  }
  clearAuthToken()
}

export function refreshToken(): Promise<RefreshTokenResponse> {
  return refreshAuthTokens()
}

export async function getPlatformUser(): Promise<PlatformUser> {
  const token = await getActiveAuthToken()
  return platformRequest<PlatformUser>('/me', {
    headers: { Authorization: `Bearer ${token}` }
  })
}

export async function updateAccount(data: { email?: string; password?: string }): Promise<PlatformUser> {
  const token = await getActiveAuthToken()
  return platformRequest<PlatformUser>('/update', {
    method: 'POST', data, headers: { Authorization: `Bearer ${token}` }
  })
}

export function isAuthenticated(): boolean {
  return getAuthToken() !== null
}

export async function getPublicSettings(): Promise<PublicSettings> {
  const { data } = await apiClient.get<PublicSettings>('/settings/public')
  return data
}

export const authAPI = {
  login, register, getCurrentUser, logout, refreshToken, getPlatformUser, updateAccount,
  isAuthenticated, setAuthToken, setRefreshToken, setTokenExpiresAt, getAuthToken,
  getRefreshToken, getTokenExpiresAt, clearAuthToken, getPublicSettings
}
export default authAPI
