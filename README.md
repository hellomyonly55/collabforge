# CollabForge

**简体中文** | [English](README.en.md)

**Team Collaboration & Project Management Platform**

CollabForge 是一个团队项目协作平台，提供团队与成员管理、项目看板、任务分配、评论、活动记录和统计分析，帮助团队跟踪工作进度并管理协作流程。

后端采用 Go、Gin、GORM、MySQL 和 Redis，前端采用 Vue 3、TypeScript、Element Plus 和 Pinia，支持通过 Docker Compose 部署。

## 功能特性

| 模块 | 能力 |
|---|---|
| 认证 | 注册、登录/退出、个人资料、改密、bcrypt、Redis 会话与限流 |
| 团队 | 创建/编辑、邀请码、成员角色、退出/转让、归档 |
| 项目 | 创建/编辑/归档，继承当前团队成员权限 |
| 任务 | 四列看板、拖拽与按钮排序、单负责人、优先级、日期、筛选、我的任务 |
| 协作 | 纯文本评论增删改、团队/项目日志检索、乐观锁冲突 |
| 统计 | 状态/优先级/负责人分布、完成率与逾期统计 |
| 后台 | 安全初始化管理员、用户禁用/启用、全会话撤销、团队归档、平台审计 |
| 测试 | 集成测试、浏览器端到端测试、多账号协作流程与依赖故障恢复验证 |

## 快速启动

进入包含 compose.yaml 的项目根目录，启动 Docker Desktop / Docker 引擎，需要 Docker Compose v2。完整 Docker 模式不要求宿主机安装 Go。

有 Node.js 时，生成本机私有配置（不会覆盖已有 .env）：

```sh
node scripts/init-env.mjs
docker compose up --build -d
docker compose ps -a
curl -i http://localhost:8080/api/v1/health/ready
```

没有 Node.js 时，复制 `.env.example` 为 `.env`，手工把 MYSQL_PASSWORD 和 MYSQL_ROOT_PASSWORD 改成两个不同的非空密码，再执行 Compose 命令。已有数据卷必须保留原数据库密码，修改 .env 不会自动改库内密码。

访问 [CollabForge](http://localhost:8080)，健康页为 [系统状态](http://localhost:8080/health)。migrate 显示 Exited (0) 正常；ready 应为 200，mysql/redis/migrations 都为 ok。初次构建需要下载镜像及依赖。端口被占用时修改 .env 的 WEB_PORT、MYSQL_PORT、REDIS_PORT；本地 Go 模式还要同步 REDIS_ADDR。浏览器使用 localhost，与配置的 Origin 保持一致。

停止并保留数据：

```sh
docker compose down
# 再次启动会复用 MySQL/Redis 数据卷：
docker compose up -d
```

当前数据库迁移版本为 **12**，已有环境可执行 `docker compose up --build -d` 升级。

## 注册与管理员

用户注册后即可登录，并创建团队或通过邀请码加入团队。

新环境初始化管理员：

```sh
docker compose exec backend /app/admin --email your-admin@example.com --nickname 管理员
```

替换为未注册邮箱，按终端隐藏提示设置密码两次；无默认管理员密码、不覆盖已有注册邮箱。自动化支持明确的 --password-stdin 管道，不接受命令行密码参数。

管理员凭据仅由部署者在本机保管，不应提交到 Git。新环境不会自动生成管理员账号。管理员登录后左侧出现“管理员后台”。

## 开发模式

需要 Go 1.26+、Node.js 22.12+（Docker 前端使用 Node 24）、npm、运行中的 Docker。

```sh
# 项目根目录；仅加载自己维护的可信 .env。
set -a
. ./.env
set +a
docker compose up -d mysql redis
cd backend
go mod download
go run ./cmd/migrate
go run ./cmd/server
```

另开终端，在项目根目录：

```sh
cd frontend
npm ci
npm run dev
```

浏览器访问 http://localhost:5173，Vite 将 /api 同源代理到 127.0.0.1:8081。Compose 内后端不向宿主机暴露业务端口。本地 Go 不自动读取 .env。

## 测试与验证

先安装前端依赖和测试浏览器：

```sh
cd frontend
npm ci
npx playwright install chromium
cd ..
./scripts/verify.sh
```

已有系统 Chrome 可替代下载，例如 macOS：

```sh
PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' ./scripts/verify.sh
```

验证脚本依次执行私有配置初始化测试、Compose 校验、Go 单元/静态检查/构建、OpenAPI 与文档校验、前端类型与构建、集成测试、浏览器回归、多账号协作流程与故障恢复验证。需要宿主机 Go、Node、Docker；失败即非零退出。

也可分别运行：

```sh
(cd backend && go test ./... && go vet ./... && go run ./cmd/checkdocs)
(cd frontend && npm run build)
./scripts/test-integration.sh
./scripts/test-e2e.sh
./scripts/test-workflow.sh
```

Go 集成使用独立 taskflow_test/Redis（13306/16379）；浏览器测试使用独立的 18080 应用端口及 MySQL/Redis 环境，测试后停止。三账号流程另启一套全新测试状态，以保留实际注册限流而不污染主库。E2E_WEB_PORT 可修改端口。直接 npm run test:e2e 会在所选目标写真实数据；完整验证使用脚本，管理员场景需要脚本的 CLI 初始化。

普通 go test 未设置 TASKFLOW_INTEGRATION 时明确跳过外部依赖测试。脚本失败时查看退出码、最后的失败用例及 frontend/test-results 中的错误上下文；测试覆盖说明见 [测试矩阵](docs/test-matrix.md)。

## 业务规则与权限

- 团队 owner/admin/member 与平台 admin/user 分开判断；平台管理员不自动获得团队任务权限。
- 所有资源查询校验当前成员和所属团队。成员退出/被移除后清空该团队未删除任务分配；owner 必须先转让再退出。
- 团队/项目归档后保留历史、普通业务只读；软删除任务不进列表/统计。评论作者可编辑，作者或团队管理员可删除。
- 任务与评论使用独立 version。排序会递增受影响任务版本；看板分页用项目 revision 避免拼接不同快照。
- 业务变更与对应日志同事务；禁用/启用递增数据库会话版本，旧会话不会恢复，即使 Redis 清理失败也无法绕过。
- ID 输出为字符串；审计时间 UTC，截止日期 DATE；逾期按 Asia/Shanghai 当天，今日截止或已完成不算逾期。
- 会话固定 7 天，HttpOnly/SameSite Cookie；改密撤销全部设备会话，凭证不放 localStorage。没有实时推送，其他成员更新后手动刷新或按冲突提示重试。

## 文档入口

- [架构与核心 Go 文件](docs/architecture.md)、[数据库](docs/database.md)、[完整 API](docs/openapi.yaml)
- [功能与测试覆盖](docs/test-matrix.md)

## 常见问题与功能范围

Docker daemon 不可用时先启动 Docker；配置缺失时检查 .env 和 shell 加载；502 检查 backend/Nginx；ready=503 检查依赖与迁移。Origin=403 时使用配置中的同源地址，避免在 localhost 与 127.0.0.1 间混用。429 表示请求触发限流，按提示等待；集成/浏览器测试使用独立环境。旧页面在更新后动态资源加载失败时刷新整页。

当前版本暂不支持实时推送、邮件、附件、楼层回复、Markdown、自定义流程、趋势历史、导出、恢复归档/删除、在线提权或管理员重置他人密码。平台审计已入库，暂无检索页面。负责人统计一次返回全部分组，整列重排适合中小项目。
