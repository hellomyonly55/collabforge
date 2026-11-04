# 阶段 1：用户认证验收

本页保留阶段 1 的交付边界；团队现已实现，当前验收见 [阶段 2](acceptance-stage-2.md)。

## 已实现接口

| 方法与路径（前缀 /api/v1） | 请求 | 成功结果 |
|---|---|---|
| POST /auth/register | email、nickname、password | 201 用户 DTO，不自动登录 |
| POST /auth/login | email、password | 200 用户 DTO，设置会话 Cookie |
| POST /auth/logout | 无 | 204，撤销当前会话并清 Cookie |
| GET /me | 登录 Cookie | 200 当前用户 DTO |
| PATCH /me | nickname | 200 更新后的用户 DTO |
| PUT /me/password | old_password、new_password | 204，撤销所有会话并清 Cookie |

用户 DTO 仅包含 id（字符串）、email、nickname、platform_role、created_at。错误含 error.code/message/request_id；400 参数错误，401 凭据错误/会话失效，409 邮箱重复/版本冲突，429 限流，503 MySQL/Redis 故障。未知字段和超过 16 KiB 的 JSON 请求体被拒绝。

## 启动、升级与停止

```sh
cd /Users/hj/Desktop/Go/TaskFlow
docker compose up --build -d
docker compose ps -a
```

既有阶段 0 数据卷保留，迁移服务自动执行 000002_users.sql。migrate 为 Exited (0) 正常。访问 http://localhost:8080；系统状态页移至 http://localhost:8080/health。

本地 Go/Vite 运行方式见 README：加载可信 .env，先运行 go run ./cmd/migrate，再分别启动 Go 和 Vue。APP_ORIGIN 必须匹配浏览器来源，本地开发使用 http://localhost:5173。HTTPS 来源自动启用 Secure Cookie，不能用 HTTP 访问该配置。

停止并保留演示数据：docker compose down。单独停止测试服务：docker compose --profile test stop mysql-test redis-test。

## 浏览器验收

1. 未登录打开 /settings，应跳转 /login；/health 无需登录。
2. 注册昵称、邮箱及密码；密码必须至少 8 个字符，且 UTF-8 不超过 72 字节；成功进入登录页，不自动登录。
3. 用同一邮箱（包括仅大小写不同）重复注册，应明确提示已注册；短密码、空昵称、确认密码不一致不能通过。
4. 错误密码显示统一凭据错误；正确密码进入团队空间。团队功能标记阶段 2 待开发，不展示假列表。
5. 刷新工作区仍保持登录；个人设置可修改昵称，刷新后仍为新昵称。邮箱只读。
6. 在另一个浏览器/无痕窗口登录相同账号，形成两个独立会话。
7. 输入错误原密码修改密码，应被拒绝且仍可使用当前会话。
8. 输入正确原密码和不同的新密码，成功后跳登录；另一个窗口刷新也应跳登录。旧密码不能登录，新密码可登录。
9. 退出登录后，刷新或直接访问 /settings 不能进入。退出只影响当前会话。
10. 开发者工具 Cookie 中 taskflow_session 应为 HttpOnly、SameSite=Lax、Path=/、7 天到期；JavaScript document.cookie 不可读到该凭证，localStorage 无会话凭证。

## 自动测试

```sh
cd /Users/hj/Desktop/Go/TaskFlow/backend
go test ./...
go vet ./...
go build ./cmd/server ./cmd/migrate
cd ..
./scripts/test-integration.sh
```

集成脚本启动 profile=test 的独立 MySQL/Redis，使用 13306/16379 端口、taskflow_test 数据库及随机 Redis 命名空间，不使用演示数据库。认证集成测试拒绝在数据库名不以 _test 结尾的环境运行。可通过 TEST_MYSQL_PORT/TEST_REDIS_PORT 修改测试端口。脚本需要 PATH 中有 go。

覆盖：注册及密码摘要、邮箱唯一、角色字段注入拒绝、用户 DTO、Cookie 标志、登录轮换、资料持久化、退出、双会话撤销、原密码校验、旧密码失效、禁用账号拒绝、未知凭证、会话到期、数据库版本失效、并发密码更新仅一个成功、登录账号/IP 和注册 IP 限流、来源验证、迁移幂等。

前端：

```sh
cd /Users/hj/Desktop/Go/TaskFlow/frontend
npm ci
npm run build
npx playwright install chromium
npm run test:e2e
```

需要应用已运行；可用 TASKFLOW_E2E_URL 指定目标。若使用本机 Chrome，可跳过浏览器下载：

```sh
PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' npm run test:e2e
```

浏览器测试会在目标环境创建随机邮箱的真实测试账号，不使用假接口，账号作为真实记录保留。测试运行使用唯一输出目录，不批量清理已有文件。不要对不允许创建测试账号的环境执行它。

## 依赖故障与恢复

停止 Redis 后，现有登录请求访问 /me 应返回 503；界面应提供错误或重试入口，不能把服务故障提示为密码错误。登录、注册和修改密码的限流依赖 Redis，因此不可用时不会跳过限流继续执行。

```sh
docker compose stop redis
# 验证浏览器与 /api/v1/health/ready
# 完成后恢复：
docker compose start redis
```

MySQL 故障同样返回 503，不能创建用户或签发成功会话。日志、响应和截图不得输出原始密码、密码摘要或凭证。

## 限流与边界

- 注册：每 IP 每小时 20 次。
- 登录：每 IP 每 15 分钟 60 次，并且每规范化邮箱每 15 分钟 10 次；包含成功与失败尝试。
- 修改密码：每用户每 15 分钟 5 次。
- 会话固定 7 天，不自动续期；Redis 重启丢失会话时需重新登录。
- Redis 清理若在密码落库后失败，session_version 仍使所有旧凭证失效，日志记录延后清理。
- 当前不含邮箱验证、找回密码、邮箱变更、管理员界面；用户禁用仅在独立测试数据库中验证，后台阶段再开放。
