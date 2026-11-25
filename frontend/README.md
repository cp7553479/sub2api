# sub2api 前端

Vue 3 / TypeScript / Pinia / Vite。平台统一管理账户、密码和会话；sub2api 保留业务账户、API key、分组和用量。

## 配置

复制 `.env.example` 并设置 `VITE_PLATFORM_API_URL` 为浏览器可访问的平台 API 地址。该 API 提供 `/auth/login`、`signup`、`refresh`、`logout`、`me` 和 `update`；前端构建必须提供此值，不能填写仅容器内部可达的主机名。

业务 API 继续使用 `VITE_API_BASE_URL`（默认 `/api/v1`）。本地开发可通过 `VITE_DEV_PROXY_TARGET` 指向业务后端。公开构建配置中不得放入数据库密码或平台管理密钥。

## 开发与验证

使用 pnpm 9 和本目录的锁文件；位于父仓库中时用 `--ignore-workspace` 隔离安装。

```bash
pnpm install --ignore-workspace --frozen-lockfile
pnpm --ignore-workspace dev
pnpm --ignore-workspace typecheck
pnpm --ignore-workspace test:run
pnpm --ignore-workspace lint:check
pnpm --ignore-workspace build
```

登录后以平台 access token 获取 `/api/v1/auth/me` 的业务 User；平台用户 UUID 与业务 bigint ID 分开。未完成业务准入时不保留浏览器登录态。注册可能要求邮件确认；退出等待平台实际撤销会话。邮箱和密码通过平台更新；业务头像、通知邮箱与业务资料继续由 sub2api 管理。

本地用户 OAuth 绑定、Passkey、TOTP、管理 MCP 授权、管理员密钥和安装建号入口已移除。供应商订阅 OAuth、微信支付回调和业务 API key 保持原生流程。
