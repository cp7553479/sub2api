import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { RouteRecordRaw } from 'vue-router'

const { state, app, harness } = vi.hoisted(() => ({
  state: { checkAuth: vi.fn(), isAuthenticated: false, isAdmin: false, isSimpleMode: false },
  app: { siteName: 'Sub2API', backendModeEnabled: false, publicSettingsLoaded: true, cachedPublicSettings: {} },
  harness: { guard: null as null | ((to: object, from: object, next: ReturnType<typeof vi.fn>) => Promise<void>), routes: [] as RouteRecordRaw[] }
}))
vi.mock('vue-router', () => ({
  createWebHistory: () => ({}),
  createRouter: (config: { routes: RouteRecordRaw[] }) => {
    harness.routes = config.routes
    return { beforeEach: (guard: typeof harness.guard) => { harness.guard = guard }, afterEach: vi.fn(), onError: vi.fn() }
  }
}))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => state }))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('@/stores/adminSettings', () => ({ useAdminSettingsStore: () => ({ customMenuItems: [] }) }))
vi.mock('@/stores/adminCompliance', () => ({ useAdminComplianceStore: () => ({ initialized: true }) }))
vi.mock('@/composables/useNavigationLoading', () => ({ useNavigationLoadingState: () => ({ startNavigation: vi.fn(), endNavigation: vi.fn() }) }))
vi.mock('@/composables/useRoutePrefetch', () => ({ useRoutePrefetch: () => ({ triggerPrefetch: vi.fn() }) }))
vi.mock('@/router/title', () => ({ resolveRouteDocumentTitle: () => 'Sub2API' }))

async function navigate(path: string, meta: Record<string, unknown>) {
  await import('@/router')
  const next = vi.fn()
  await harness.guard!({ path, fullPath: path, meta }, {}, next)
  return next
}

describe('actual route authorization', () => {
  beforeEach(() => {
    state.isAuthenticated = false
    state.isAdmin = false
    state.isSimpleMode = false
    app.backendModeEnabled = false
  })
  it('redirects unauthenticated business requests to login', async () => {
    const next = await navigate('/keys', { requiresAuth: true })
    expect(next).toHaveBeenCalledWith({ path: '/login', query: { redirect: '/keys' } })
  })
  it('refuses administrator routes to ordinary users', async () => {
    state.isAuthenticated = true
    const next = await navigate('/admin/users', { requiresAdmin: true })
    expect(next).toHaveBeenCalledWith('/dashboard')
  })
  it('allows a current administrator to reach business management', async () => {
    state.isAuthenticated = true
    state.isAdmin = true
    const next = await navigate('/admin/users', { requiresAdmin: true })
    expect(next).toHaveBeenCalledWith()
  })
  it('removes native identity/setup/MCP routes while retaining payment OAuth', async () => {
    await import('@/router')
    const paths = harness.routes.map((route) => route.path)
    for (const retired of ['/setup', '/email-verify', '/auth/callback', '/auth/wechat/callback', '/oauth/mcp/authorize', '/settings/mcp']) {
      expect(paths).not.toContain(retired)
    }
    expect(paths).toContain('/login')
    expect(paths).toContain('/register')
    expect(paths).toContain('/auth/wechat/payment/callback')
  })
})
