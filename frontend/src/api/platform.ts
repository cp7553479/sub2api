import axios, { type AxiosRequestConfig } from 'axios'

export function getPlatformAPIURL(): string {
  const url = import.meta.env.VITE_PLATFORM_API_URL?.trim()
  if (!url) throw new Error('VITE_PLATFORM_API_URL is required')
  return url.replace(/\/$/, '')
}

export async function platformRequest<T>(path: string, config: AxiosRequestConfig = {}): Promise<T> {
  try {
    const { data } = await axios.request<T>({
      ...config,
      url: `${getPlatformAPIURL()}/auth${path}`,
      timeout: 30_000,
      headers: { 'Content-Type': 'application/json', ...config.headers }
    })
    return data
  } catch (error) {
    if (axios.isAxiosError(error)) {
      const body = error.response?.data
      error.message = body?.error?.message || body?.error_description || body?.msg || body?.message || error.message
    }
    throw error
  }
}

export interface PlatformUser {
  id: string
  email?: string
}

export interface PlatformSession {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
  user: PlatformUser
}
