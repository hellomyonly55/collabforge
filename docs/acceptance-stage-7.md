# 阶段 7：管理员后台验收

> 此为历史阶段记录。TaskFlow 1.0 最终运行和验收见 [阶段 8 验收](acceptance-stage-8.md)。

## 启动与迁移

```sh
cd /Users/hj/Desktop/Go/TaskFlow
docker compose up --build -d
docker compose ps -a
curl -i http://localhost:8080/api/v1/health/ready
```

访问 http://localhost:8080。本阶段追加迁移 12：platform_audit_logs；无新增必填环境变量。migrate 为 Exited (0)，ready 为 200，三个依赖为 ok。已有账号、团队、项目、任务、评论与团队日志保留。

本地开发沿用 README：加载可信 .env，启动 MySQL/Redis，backend 先 go run ./cmd/migrate，再 go run ./cmd/server，frontend 执行 npm ci、npm run dev。停止并保留业务数据：`docker compose down`。

## 初始化管理员

新环境不包含管理员和默认密码。运行 CLI 创建一个新的管理员账号；不能把已有注册邮箱直接提权，也不覆盖原账号或密码。

```sh
# 替换为尚未注册的邮箱、你的昵称；终端输入密码两次，不回显。
docker compose exec backend /app/admin --email your-admin@example.com --nickname 管理员
```

本地 Go 模式（已加载 .env 并迁移）：

```sh
cd /Users/hj/Desktop/Go/TaskFlow/backend
go run ./cmd/admin --email your-admin@example.com --nickname 管理员
```

密码沿用 8 字符以上、最多 72 字节及 bcrypt 规则。密码不作为 CLI 参数、环境默认值或日志字段；自动化可用 --password-stdin 从受控管道输入一行密码，终端直接使用该选项会被拒绝以防回显。CLI 不自动执行迁移。重复邮箱、无效密码、迁移缺失、数据库/审计异常均非零退出。

本次本地演示若尚无管理员，会通过同一命令创建随机凭据，保存在 `.local/admin-credentials.txt`（目录 0700、文件 0600，已排除 Git 和 Docker 构建上下文）。不要将该文件提交或分享；新环境仍应自己设置密码。已有管理员时不会覆盖账号。

登录管理员账号后，左侧出现“管理员后台”。页面地址 /admin、/admin/users、/admin/teams；普通用户直接进入会看到无权限页，后端同样拒绝，不依赖隐藏菜单。

## 内容与核心 Go 文件

- `cmd/admin/main.go`：交互隐藏输入或非终端 stdin 读取密码，校验迁移，调用初始化服务，不输出密码。
- `internal/service/admin_service.go`：平台角色、当前会话版本、禁用自己限制、状态变更/归档事务、CLI 创建新管理员。
- `internal/repository/admin_repository.go`：参数化用户/团队列表、全局统计、按 ID 顺序锁账号、平台审计写入。
- `internal/dto/admin.go`：明确管理响应与筛选，不返回密码哈希或会话版本。
- `internal/middleware/admin.go`：登录中间件之后检查平台 admin 角色。
- `internal/handler/admin_handler.go`、`internal/router/admin_routes.go`：分页/ID/JSON 解析及五个管理 API。
- `internal/model/platform_audit.go`、`migrations/000012_platform_audit_logs.sql`：独立平台审计，不伪造所属团队。
- `tests/integration/admin_test.go`：初始化、安全边界、会话撤销、回滚、并发处置及 Redis 清理失败测试。

前端 `AdminView.vue` 展示真实统计、分页用户/团队列表、原因表单与操作确认；Vue 路由守卫和后端分别实施权限。完整原理见 [架构说明](architecture.md)。

## API

统一前缀 /api/v1，仅活跃平台 admin；未登录 401、普通用户 403、无效参数 400、资源不存在 404、归档/并发冲突 409、依赖故障 503。完整字段见 [OpenAPI 0.8.0](openapi.yaml)。

| 接口 | 行为 |
|---|---|
| GET /admin/statistics | 全局用户、团队、项目、任务、评论及归档/完成/逾期数量 |
| GET /admin/users | keyword 匹配邮箱/昵称；status=all/active/disabled；role=all/admin/user |
| PATCH /admin/users/:userId/status | status=active/disabled，reason 必填 1—500 字符；200 用户 DTO |
| GET /admin/teams | keyword 匹配名称/简介；status=all/active/archived |
| POST /admin/teams/:teamId/archive | reason 必填 1—500 字符；204；重复归档 409 |

列表默认 page=1/page_size=20，最大 100。百分号/下划线按字面匹配；ID 输出字符串；用户响应包含拥有团队数量，团队响应包含 owner 资料及成员/项目/未删除任务数量。管理团队列表不提供任务正文，也不授予团队任务编辑权。

平台统计的 admins 为活跃管理员数；任务和评论排除软删除，包含归档项目。逾期按上海当天、非 done 且日期早于当天。全局统计在同一读事务快照查询，不用假数据或缓存。

## 一致性与安全规则

1. 禁用自己被拒绝；请求执行时重新校验管理员状态和 session_version。
2. 状态实际改变时递增 MySQL session_version，然后撤销该用户 Redis 全部会话。即使 Redis 清理失败，数据库版本和状态仍拒绝旧凭证；重新启用再次递增版本，旧会话不会恢复，必须重新登录。
3. 登录与禁用并发时，新建的旧版本会话也会被后续鉴权拒绝。已在执行的请求按其原有事务规则完成，禁用不承诺回滚之前已开始的业务操作。
4. 禁用 owner 不转让所有权、不移除成员、不清空任务分配；启用后原角色保留。页面明确提示。
5. 两个管理员交叉禁用时按账号 ID 锁定、在锁内重新验证操作者，不能把彼此都成功禁用；至少保留一个有效操作者。没有公开修改平台角色 API。
6. 状态与平台审计同事务；平台归档、平台审计和团队日志同事务。任一步失败整体回滚，不留下成功事件。相同状态重复请求不增版本或重复记录审计。
7. 平台归档保留 owner 与历史；普通成员之后只能读，平台管理员仍不会自动成为团队成员。平台审计不存密码、凭据或哈希，不开放编辑/删除接口。

## 自动检查

```sh
cd /Users/hj/Desktop/Go/TaskFlow/backend
go test ./...
go vet ./...
go build ./cmd/server ./cmd/migrate ./cmd/admin
cd ..
./scripts/test-integration.sh
```

真实集成测试连接独立 taskflow_test 与 Redis；普通 go test 未设置开关时明确跳过外部依赖。新增覆盖 CLI 核心初始化/bcrypt/碰撞、普通用户隔离、字段防泄露、输入验证、分页筛选、实际平台计数、双会话禁用/启用、owner 保留、平台归档、审计回滚、并发交叉禁用、Redis 清理失败后的版本权威。

```sh
cd /Users/hj/Desktop/Go/TaskFlow/frontend
npm ci
npm run build
npx playwright install chromium
cd ..
./scripts/test-e2e.sh
# 只运行管理员模块：
./scripts/test-e2e.sh admin.spec.ts
```

已有 Chrome 可用 `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' ./scripts/test-e2e.sh`。脚本在独立 18080 环境调用真实 CLI 创建随机管理员，通过真实注册接口创建普通账号；临时凭据文件权限 0600，结束仅删除该一个文件并停止测试服务。统计测试复用该普通账号，不降低生产注册限流。直接 npm run test:e2e 缺少初始化变量时会明确跳过管理员场景；完整验证使用脚本。

## 可执行 API 示例

先使用自己的管理员账号登录，Cookie 文件不要提交：

```sh
COOKIE=/tmp/taskflow-stage7.cookies
curl -fsS -c "$COOKIE" -H 'Content-Type: application/json' \
 -d '{"email":"替换为管理员邮箱","password":"替换为你的密码"}' \
 http://localhost:8080/api/v1/auth/login
chmod 600 "$COOKIE"
curl -fsS -b "$COOKIE" http://localhost:8080/api/v1/admin/statistics
curl -fsS -b "$COOKIE" 'http://localhost:8080/api/v1/admin/users?page=1&page_size=20'
curl -fsS -b "$COOKIE" 'http://localhost:8080/api/v1/admin/teams?status=active'
```

替换成用于验收的真实用户 ID；会实际禁用账号：

```sh
USER_ID=替换为测试账号ID
curl -i -b "$COOKIE" -X PATCH -H 'Content-Type: application/json' \
 -d '{"status":"disabled","reason":"验收多会话撤销"}' \
 "http://localhost:8080/api/v1/admin/users/$USER_ID/status"
# 验收完成后恢复登录能力，旧会话仍不可用：
curl -i -b "$COOKIE" -X PATCH -H 'Content-Type: application/json' \
 -d '{"status":"active","reason":"验收完成"}' \
 "http://localhost:8080/api/v1/admin/users/$USER_ID/status"
```

## 浏览器验收标准

1. 新管理员由 CLI 创建，密码输入不回显。重复邮箱失败，不修改已有用户角色；数据库保存 bcrypt 哈希，API 无密码/哈希/会话版本。
2. 普通用户访问 /admin 被拒绝，直接调用五个后台 API 也为 403；未登录为 401。
3. 管理员看到真实平台数量；用户/团队关键词、状态、角色及分页结果准确。空结果、加载和故障不使用模拟数据。
4. 普通用户两个浏览器登录，管理员填写原因并禁用：两端下一次鉴权均 401，密码正确也不能登录；启用后旧会话仍 401，重新登录成功。
5. 用户为团队 owner 时禁用不会转让 owner、丢失团队或任务分配；恢复后角色保留。当前管理员自身禁用按钮不可用，直接请求也被拒绝。
6. 管理员能从后台看到非成员团队摘要，但直接访问其任务/项目仍 404；不得获得普通业务编辑权。
7. 填原因确认归档团队后，owner 和历史保持；成员创建/编辑任务或项目被拒绝，团队动态出现平台管理员操作及原因。
8. 日志失败测试证明状态/版本、归档和两类日志全部回滚；并发交叉禁用不能让两个操作者都成功禁用。
9. 390px 后台正常显示，表格局部滚动，整页不横向溢出。

## 限制与下一阶段

没有管理员在线提权、密码重置、解封归档、物理删除或平台审计检索页面；审计已真实持久化，可在数据库检查。初始管理员凭据需自己保管。前后端不使用默认演示管理员密码。

阶段 8 将完善最终回归、双普通账号和管理员完整演示、README 与校招项目交付说明。
