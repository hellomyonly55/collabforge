# 阶段 7 实测记录

> 此为历史阶段记录。TaskFlow 1.0 最终运行和验收见 [阶段 8 验收](acceptance-stage-8.md)。

日期：2026-10-31（Asia/Shanghai）。运行、API、管理员初始化、测试及手工验收标准见 [阶段 7 验收](acceptance-stage-7.md)。

## 已执行检查

| 检查 | 结果 |
|---|---|
| Go 1.27.3，go test ./...、go vet ./... | 通过；普通运行明确跳过外部依赖测试 |
| scripts/test-integration.sh | 37 项真实 MySQL/Redis 集成测试通过，14.569 秒 |
| npm run build | Vue 类型检查、Vite 构建通过 |
| scripts/test-e2e.sh，系统 Chrome | 10 项完整浏览器回归通过，24.4 秒 |
| Docker 内 server/migrate/admin 构建 | 通过 |
| CLI 初始化及实际管理员登录 | 独立测试环境和本地演示环境均通过 |
| OpenAPI 0.8.0 | YAML 解析和内部引用校验通过 |
| docker compose up --build -d | 主环境升级成功，迁移版本 11 → 12 |
| GET /api/v1/health/ready | HTTP 200，mysql/redis/migrations 均为 ok |

## 新增 5 项真实集成测试

- TestAdminBootstrapAndAuthorization：新建管理员/bcrypt、已有邮箱不被提权、普通用户与未登录隔离、哈希/会话版本不公开、非法状态/原因/字段、自身禁用拒绝。
- TestAdminStatusRevokesAllSessionsWithoutTransferringOwnership：双会话禁用、无法登录、启用后旧会话仍无效、新登录恢复；owner 与任务分配不变；重复状态不增审计。
- TestAdminArchiveListsAndStatistics：真实用户/团队筛选与数量、实际全局用户计数、平台归档及原因日志、平台 admin 没有团队任务访问权、非法参数拒绝。
- TestAdminMutationsAuditRollbackAndConcurrentDisables：审计失败时账号状态/版本、归档、团队日志和初始化全部回滚；管理员交叉禁用只能一个成功。
- TestAdminSessionVersionAuthorityWhenRedisCleanupFails：关闭清理使用的 Redis 客户端，让数据库状态成功提交但清理失败；经另一个真实 Redis 连接验证禁用及启用后旧凭证都返回 401。

此前 32 项认证、团队、项目、任务、评论、日志、统计和迁移测试均同时回归通过。数据库和 Redis 均使用独立测试环境，未对主库注入故障。

## 浏览器与 CLI 实测

E2E 脚本在独立 18080 应用栈调用真实 /app/admin --password-stdin 创建随机管理员，再通过真实注册接口创建普通用户；没有默认管理员密码、提权测试路由或前端 mock。

新增场景完成：普通用户后台被拒绝 → 管理员登录并看到真实统计 → 筛选用户 → 填原因禁用 → 两个普通会话下一次请求均 401、密码登录也失败 → 启用后旧会话仍 401，新登录成功 → 平台查看非成员团队摘要但普通任务 API 为 404 → 填原因归档 → owner 保留且后续项目创建 409 → 390px 后台无整页横向溢出。

既有 9 项注册、团队、项目、任务拖拽与冲突、评论/日志、统计等浏览器流程一并通过。测试结束停止独立测试服务，并只删除其单个 0600 临时凭据文件。

## 本地部署与管理员

升级前主库：users=13、platform admins=0、teams=3、projects=0、tasks=0、comments=0、activity_logs=20，迁移版本 11。

升级及初始化后：users=14、platform admins=1，其余上述业务计数保持；platform_audit_logs=1，为真实 admin.created；迁移版本 12。没有预置示例任务或修改现有用户身份。管理员通过真实 CLI 新建，登录及 GET /admin/statistics 已验证后退出测试会话。

登录信息仅保存在本机 `.local/admin-credentials.txt`，目录 0700、文件 0600，已排除 Git 和 Docker 构建上下文；密码未输出到聊天、日志或文档。文件不是新环境默认凭据，重新安装需使用 CLI 自设密码。

主应用保持 http://localhost:8080 运行，MySQL/Redis 数据卷保留，独立测试依赖已停止。

## 真实页面截图

以下来自独立真实测试账号与数据库，不是主库中的模拟记录：

- [平台统计](images/stage-7-admin-statistics.png)
- [用户禁用与筛选](images/stage-7-admin-users.png)
- [390px 团队管理](images/stage-7-admin-mobile.png)

## 限制

未提供在线平台提权、管理员重置他人密码、恢复团队归档、物理删除或平台审计检索页面；审计已写入真实数据库。禁用不会回滚已开始的业务事务，也不转让团队所有权。下一阶段 8 完成最终交付回归和完整演示文档。
