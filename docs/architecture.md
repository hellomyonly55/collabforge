# TaskFlow 架构与开发约定

## 当前交付：TaskFlow 1.0（阶段 0—8）

Vue 3 + TypeScript + Element Plus + Pinia + Vue Router → Nginx/Vite 同源代理 → Gin → Handler → Service → Repository → GORM/MySQL、Redis。

已实现工程基础、用户认证与个人设置、团队管理、项目管理、任务与看板、评论、可筛选团队/项目动态及任务统计及管理员后台。尚未提供的扩展接口返回 404。页面只使用真实后端数据。

## 核心 Go 文件

- `cmd/server/main.go`：加载配置、创建 MySQL 和 Redis 客户端、组装依赖、启动 HTTP、处理 SIGINT/SIGTERM 和优雅退出。
- `cmd/migrate/main.go`：独立运行版本化迁移，失败退出码为 1；不会由 HTTP 服务隐式修改表结构。
- `internal/config/config.go`：校验环境变量；必填项缺失时明确提示变量名，不输出密码。
- `internal/infrastructure/database/database.go`：构建安全 DSN、设置 UTC、连接池和超时；GORM 静默日志避免泄露 SQL/配置。
- `internal/infrastructure/redis/redis.go`：创建带超时的 Redis 客户端。
- `internal/repository/health_repository.go`：真实 MySQL PING、Redis PING 和迁移版本查询。
- `internal/service/health_service.go`：在总超时内汇总依赖健康状态。
- `internal/handler/health_handler.go`：转换健康状态为 HTTP 200 或 503，不暴露依赖原始错误。
- `internal/middleware/http.go`：随机请求 ID、JSON 访问日志、异常恢复、写请求来源检查。
- `internal/router/router.go`：路由组装，不信任来自任意代理的客户端 IP。
- `migrations/migrate.go`：嵌入 SQL，使用同一 MySQL 连接持有 advisory lock，避免迁移并发执行。

## 迁移约定

采用编号 SQL 文件和 schema_migrations 表，不使用 AutoMigrate。每个 SQL 文件只包含一条语句。MySQL DDL 不保证事务回滚，因此迁移必须可安全重试；变更用新文件，禁止修改已执行文件。每次新增迁移同步更新 LatestVersion。阶段 2 追加版本 3—6 的团队、成员、邀请及日志表；阶段 3 追加项目表和日志组合外键；阶段 4 新增任务表和项目看板修订号，阶段 5 新增评论表，当前迁移版本为 11。既有数据卷原地升级，不清空数据。

## 后续模块契约

- Repository 处理数据访问；Service 实施权限和事务；Handler 不直接编写业务 SQL。
- 业务修改和 activity_logs 同事务，自团队模块起记录。
- 认证模块已落地 bcrypt、Redis 会话、请求限流和前端路由守卫。
- MySQL 保存业务事实，Redis 不替代业务数据库。
- 后续采用 BIGINT ID，API 输出字符串；数据库时间 UTC，日期字段 DATE。
- 单体部署，不引入微服务、Kubernetes、消息队列。

## 请求约定

前端经同源代理访问 `/api/v1`。读取请求正常访问；写请求存在 Origin 时必须匹配 APP_ORIGIN，Sec-Fetch-Site 为 cross-site 时拒绝。命令行客户端可不带 Origin。错误不包含 SQL、密码或堆栈，响应携带 request_id 用于关联日志。健康检查 503 是诊断响应，仍返回 data.checks；普通业务错误将返回 error 对象。

参考官方文档：[Gin](https://gin-gonic.com/en/docs/quickstart/)、[GORM MySQL](https://gorm.io/docs/connecting_to_the_database.html)、[go-redis](https://redis.io/docs/latest/develop/clients/go/)。

## 阶段 1 核心文件

- `model/user.go`：用户数据库实体，所有字段禁止直接 JSON 序列化。
- `dto/auth.go`：注册、登录、资料和密码请求；显式公开用户 DTO，ID 转字符串。
- `infrastructure/password/password.go`：8 字符/72 字节校验、bcrypt 哈希和密码比较。
- `repository/user_repository.go`：唯一邮箱、资料更新、密码与 session_version 原子更新。
- `repository/session_repository.go`：随机 256 位凭证，SHA256 Redis 键；会话 TTL、用户会话索引、原子撤销和限流。
- `service/auth_service.go`：输入归一化、注册、登录、鉴权、改密和失效规则。
- `middleware/auth.go`：登录验证、IP/用户限流，Redis 异常时拒绝执行认证操作。
- `handler/auth_handler.go`：16 KiB JSON 限制、未知字段拒绝、Cookie 设置/清除、响应映射。
- `migrations/000002_users.sql`：新增 users，不改写已应用的阶段 0 迁移。

密码修改将 MySQL 的 session_version 加一，每次鉴权同时比较数据库版本与 Redis 会话版本。因此即使改密后的 Redis 清理失败，旧凭证仍会被拒绝。Redis 清理失败只记录不含凭证的警告，密码修改结果仍为成功。并发改密使用旧版本条件更新，仅允许一个请求成功。

会话固定 7 天，不自动续期。登录成功生成新凭证，并撤销当前浏览器之前的会话。退出仅撤销当前会话；修改密码撤销所有设备。会话凭证不写 localStorage、不进入 JSON 和访问日志。登录失败对不存在、密码错误及禁用账号使用统一响应，并进行同成本的 bcrypt 比较。

默认不信任代理；Compose 后端只从内部网络访问，显式信任 Docker 私有网段，由 Nginx 覆盖 X-Real-IP。后续公网部署时应按实际入口收窄 TRUSTED_PROXIES。Secure Cookie 自动由 APP_ORIGIN 的 HTTPS 协议决定；当前本地 HTTP 使用非 Secure Cookie。

前端仅将用户资料保存在 Pinia 内存；刷新后请求 /me 恢复登录。401 清空用户状态并转登录页；503/网络故障显示连接错误页，保留重试入口，避免将服务故障误认为密码错误。

参考：[bcrypt 官方文档](https://pkg.go.dev/golang.org/x/crypto/bcrypt)。

## 阶段 2 核心 Go 文件

- `internal/model/team.go`：团队、软移除成员、邀请码摘要和操作日志实体。
- `internal/dto/team.go`：团队/成员/邀请/日志响应及泛型分页；所有 ID 显式转字符串，邀请码摘要不进入响应。
- `internal/repository/team_repository.go`：团队事务、FOR UPDATE 锁、成员查询、邀请计数、日志持久化及分页；事务绑定的 Repository 保证全部查询使用同一连接。
- `internal/service/team_service.go`：角色边界、归档只读、邀请码验证、成员生命周期、所有权转让；每次写入同步记录 activity_logs。
- `internal/handler/team_handler.go`：正整数 ID、分页和 JSON 参数解析，转换为统一响应。
- `internal/router/team_routes.go`：注册登录保护后的团队路由，配置创建与加入限流。
- `migrations/000003_teams.sql` 至 `000006_activity_logs.sql`：版本化新增表、外键、唯一索引及 CHECK 约束。

所有团队写入遵守“锁团队 → 读取当前角色 → 校验业务 → 修改数据 → 写日志 → 提交”的固定顺序。转让时先将旧 owner 设为 admin，再将目标成员设为 owner，最后同步 owner_id；任一步失败整体回滚。数据库生成列唯一索引阻止同队出现多个活跃 owner，服务事务确保至少一个。

加入团队先用摘要定位邀请，再在团队锁内重新读取成员和邀请状态。首次成功同时恢复/创建 member、增加使用次数和记录日志；重复加入不重复执行。移除/退出只设置 removed_at；重新加入更新 joined_at 并恢复普通成员身份。

前端团队页使用独立 API 模块和显式类型。详情请求用 generation 计数防止旧路由/旧标签页响应覆盖新页面；权限丢失后清空旧数据与邀请码。转让、移除、归档和退出提供明确确认，服务端始终再次校验权限。

参考：[GORM 事务](https://gorm.io/docs/transactions.html)、[GORM 行锁](https://gorm.io/docs/advanced_query.html)、[MySQL 生成列](https://dev.mysql.com/doc/refman/8.4/en/create-table-generated-columns.html)。

## 阶段 3 核心 Go 文件

- `internal/model/project.go`：项目实体，保存所属团队、创建者、资料及归档时间。
- `internal/dto/project.go`：项目请求与响应，ID 转字符串，显式附带当前团队角色和父团队归档状态。
- `internal/repository/project_repository.go`：GORM 项目查询、分页、创建、修改和项目行锁。
- `internal/repository/activity_repository.go`：团队与项目共享的日志写入和按范围分页，使用调用方的事务连接；团队原有日志行为保留。
- `internal/service/project_service.go`：在团队权限边界内管理项目，检查父团队/项目归档，保证业务与日志同事务。
- `internal/handler/project_handler.go`：复用 ID、分页和 JSON 校验，输出稳定 HTTP 错误与 DTO。
- `internal/router/project_routes.go`：登录保护的项目路由及创建限流。
- `migrations/000007_projects.sql`、`000008_activity_project_fk.sql`：项目表、索引、外键和日志跨团队一致性约束。

项目所属团队不可修改。直接按项目 ID 请求时，先定位父团队，再在同一事务中获取当前成员关系；写操作锁团队后再锁项目，与成员移除、角色管理和团队归档保持相同锁序。创建者身份不替代当前团队角色，不引入项目成员权限表。

归档项目只修改自身 archived_at；归档团队通过父状态让所有子项目只读，不批量改写项目状态。项目动态按 team_id/project_id 共同限定范围，团队动态包含该团队的项目事件。

MySQL ALTER DDL 无法与版本记录形成原子提交。迁移 8 在执行前检查 information_schema 中外键的两列关联；DDL 成功但版本记录中断时可跳过已完成 DDL 并补记版本。后续 ALTER 迁移也应定义精确后置条件。

浏览器回归由 compose.e2e.yaml 启动独立的真实栈；scripts/test-e2e.sh 等待健康、执行 Playwright 并停止测试服务，避免污染演示账号和限流计数。Go 集成测试仍使用独立 profile=test 服务。

## 阶段 4 核心 Go 文件与请求路径

- `internal/model/task.go`：任务持久化实体；可空负责人/截止日期、软删除、顺序和乐观锁版本，不直接序列化实体。
- `internal/dto/task.go`：请求、任务响应、列表和看板结构；`NullableString` 区分 PATCH 未提供字段与显式 null，允许真正清空日期/负责人。
- `internal/handler/task_handler.go`：解析任务 ID、分页、筛选、版本和 JSON，请求无效返回稳定 400，不在 Handler 编写业务 SQL。
- `internal/service/task_service.go`：当前成员权限、字段规则、单负责人校验、归档只读、创建/编辑/删除/移动、版本冲突和日志事务。`taskToday` 明确采用上海日历日期。
- `internal/repository/task_repository.go`：GORM 参数化查询、真实关联名称、筛选、排序、版本条件更新、项目 revision 和离队分配清理。
- `internal/service/project_service.go`：共用 `inProjectAccess` 权限与事务入口；项目管理要求 owner/admin，任务写入允许普通成员，两者都检查归档和当前成员关系。
- `internal/repository/team_repository.go`：软移除成员前调用 `ClearAssignments`；清理任务、任务日志和成员变更任一步失败一起回滚。
- `internal/router/task_routes.go`：注册登录保护后的任务、看板与我的任务路由；创建任务每用户每小时最多 300 次。
- `migrations/000009_tasks.sql`：任务表、外键、枚举 CHECK 约束、看板/负责人/优先级/日期索引。
- `migrations/000010_project_task_revision.sql`：项目任务快照计数；迁移工具检查该列类型、非空和默认值，支持 DDL 成功但版本记录中断后的恢复。
- `tests/integration/task_test.go`：独立真实 MySQL/Redis 的权限、并发排序、清理、回滚、分页快照和迁移测试。

例如移动任务：Gin 登录中间件 → TaskHandler.Move 解析 version/status/before_task_id → TaskService 验证当前成员与归档 → Repository 在同一事务锁团队、项目、任务 → 校验版本和插入目标 → 更新旧/新列连续位置及发生变化的任务版本 → 增加项目 revision → 写 task.moved 日志 → 提交。SQL/日志失败回滚所有变化；跨团队目标不会被接受。

所有任务写入固定锁序“团队 → 项目 → 任务”，与成员移除、项目归档共用父锁。普通编辑带 version 条件更新；排序影响到其他任务时，也增加其版本。无变化移动不会新增日志或递增版本，但仍检查客户端版本。删除软标记并压紧原列位置；同样校验 version。列表排除软删除行。

每列 50 条的分页不能拼接不同快照。看板读取在同一事务返回四列与 revision；加载后续列页传入该 revision，项目内容、排序、分配或删除变化后返回 409/BOARD_CHANGED。前端丢弃旧分页并重载整个看板。

移除/退出成员在团队锁内清空所有非删除任务的 assignee_id，并递增任务版本、受影响项目 revision，逐条记录 task.unassigned。包括已归档项目和已完成任务，这是成员一致性清理；不开放归档任务的普通编辑。清理失败会回滚成员移除及全部日志。

前端 `TaskPanel.vue` 复用到抽屉与独立详情；`TaskBoardView.vue` 调用真实 API 实现原生拖拽，失败重新拉取数据，并提供前移/后移及状态选择的操作入口。`AssigneePicker.vue` 分页读取真实团队成员，`MyTasksView.vue` 只读取当前用户有权限的任务。请求有 generation 防止旧路由响应覆盖新页面；不使用假接口、定时模拟或 localStorage 业务数据。

首版按整列遍历重排，适合校招演示和中小团队；超大项目需要后续优化排序策略。ID 按字符串返回，version/revision 是数字计数。日期不含时区；审计时间保持 UTC。

参考：[GORM 更新零值及条件](https://gorm.io/docs/update.html)、[MySQL DATE](https://dev.mysql.com/doc/refman/8.4/en/datetime.html)、[原生拖放 API](https://developer.mozilla.org/en-US/docs/Web/API/HTML_Drag_and_Drop_API)。

## 阶段 5 核心 Go 文件与请求流程

- `internal/model/comment.go`：评论实体，关联任务和作者，保留软删除标记及独立版本。
- `internal/dto/comment.go`：评论请求/公开响应，显式输出字符串 ID、当前 can_edit/can_delete；不暴露持久化实体。
- `internal/repository/comment_repository.go`：真实分页、作者昵称关联、行锁和带版本的更新/软删除。
- `internal/service/comment_service.go`：复用任务的成员/归档边界，实施作者编辑和作者/管理员删除规则，负责业务与日志事务；不调用任务更新或看板修订操作。
- `internal/handler/comment_handler.go`：任务/评论 ID、内容、分页和版本解析，返回统一 JSON/204。
- `internal/router/comment_routes.go`：评论路由、登录校验和每用户每小时 120 次的发表限流。
- `internal/dto/activity.go`：日志筛选条件，与分页分开表达，不改变已有日志响应结构。
- `internal/handler/activity_handler.go`：解析可选 action/target_type/actor_id/project_id/from/to，拒绝非法 ID、类型、时间或倒置区间。
- `internal/service/activity_service.go`：统一的操作类型与目标白名单、RFC3339 时区转换和范围校验；团队和项目日志服务复用。
- `internal/repository/activity_repository.go`：在已有 team_id/project_id 权限范围上叠加参数化筛选；筛选不能扩大访问范围；按日志 ID 倒序。
- `migrations/000011_comments.sql`：评论表、任务/作者外键、版本 CHECK 和任务分页索引。新增 CREATE 迁移支持安全重试，不改动旧版本。
- `tests/integration/comment_test.go`：真实权限、软删除、归档/离队、并发冲突、事务回滚、分页与日志筛选、限流测试。

例如编辑评论：Gin 鉴权 → Handler 校验 content/version → CommentService 根据评论定位所属任务 → 复用 TaskService.inTask 在同一事务锁团队、项目、任务 → 锁评论 → 检查作者身份、归档和版本 → 条件更新 content/version → 写 comment.updated → 提交。任一步失败一起回滚。不存在、已删除任务/评论及非成员统一 404；普通成员越权为 403；过期版本为 409/COMMENT_VERSION_CONFLICT。

评论正文 1—2000 Unicode 字符，去除前后空白后保存。首版为纯文本，Vue 插值展示，不使用 v-html。审计只记录评论 ID、所属任务 ID/标题，不复制正文；编辑或删除后的旧正文不会通过日志接口再次公开。数据库软删除仍保留历史行，不提供物理清理接口。

`CommentThread.vue` 复用到任务抽屉与独立详情；处理分页、编辑/删除、版本冲突、空列表和错误。`ActivityFeed.vue` 同时用于旧动态页与新的 `ActivityView.vue`，避免两套日志展示；请求 generation 防止旧路由响应覆盖新范围。记录链接仍由详情 API 校验权限，已删除资源会得到真实 404。

日志日期 UI 将上海日历日转换为带 +08:00 的起止 RFC3339 时间；服务端转换 UTC 后按包含边界筛选，保持 UTC 审计时间与用户日期一致。仅查询追加式日志，不开放修改/删除审计接口。首版使用分页而非实时流。

## 阶段 6 核心 Go 文件与统计口径

- `internal/dto/statistics.go`：公开统计响应、状态/优先级组、负责人组，ID 转字符串；明确日期、时区与查询时间。
- `internal/repository/statistics_repository.go`：SQL COUNT/SUM/GROUP BY 聚合；先限定团队、可选项目，再排除软删除。统计全部行，不受任务列表分页或看板每列 50 条限制。
- `internal/service/statistics_service.go`：复用当前团队成员权限与项目所属团队边界，使用同一个事务连接完成全部查询；补足四种状态/优先级零值，计算完成率，始终补充未分配组。
- `internal/handler/statistics_handler.go`：解析团队/项目 ID、调用服务，沿用统一成功和失败响应。
- `internal/router/statistics_routes.go`：注册两个只读统计 API，登录校验及 Cache-Control: no-store。
- `tests/integration/statistics_test.go`：真实数据变化、105 条全量聚合、当前权限、归档/离队、读快照及数据库失败检查。

请求流程：Gin 鉴权 → Handler 解析 ID → StatisticsService 在 TeamService.inTeam / ProjectService.inProjectAccess 中验证当前成员 → Repository 使用该事务连接查询总数、状态、优先级、负责人 → Service 补足零值及响应上下文 → 返回真实结果。

当前 MySQL 采用默认 REPEATABLE READ。权限读取与全部聚合共享一个读事务快照，避免并发写入使总数与分组相互矛盾；不会获取写锁。测试在第一项聚合后实际提交一条新任务，证明当前请求维持旧快照、下一请求可见新任务。数据库查询失败返回 503，不把失败当作全零数据。

总数排除 tasks.deleted_at 非空的行，包含已归档项目。完成仅 status=done；未完成=总数-已完成；逾期为非 done 且 due_at 早于本次请求捕获的上海当天。完成率=已完成/总数×100，四舍五入到一位小数；分母零返回 0。

负责人按 ID 聚合并带当前昵称；未分配 user_id=null，零任务也保留这一组。其他负责人没有任务时不返回，不把成员总数混作任务分布。离队清理事务完成后，相关任务自然进入未分配组。状态和优先级按固定枚举顺序返回四组，即使计数为零。

前端 `StatisticsView.vue` 同时承载团队/项目页面，CSS 条形图与环形图均来自真实统计响应，旁边保留精确数字和可访问名称；无需额外图表库。请求 generation 防止旧路由响应覆盖新页面，刷新时清空旧统计，权限/依赖错误不残留假成功状态。手机端表格在局部横向滚动。

首版无趋势历史、日期范围或缓存，不添加统计表、定时任务及分布式组件。负责人聚合全部返回，适合中小团队；超大团队后续可增加聚合分页。若未来修改 MySQL 隔离级别，需要同步维持快照保证。

## 阶段 7 核心实现

`cmd/admin/main.go` 使用 x/term 隐藏输入或明确的非终端 stdin，校验迁移后调用 BootstrapAdmin；新账号、bcrypt 和 admin.created 审计同事务，邮箱碰撞不接管既有账号。Docker 镜像新增 /app/admin，无默认账号或密码。

`middleware/admin.go` 在真实登录验证后校验平台角色；`admin_handler.go` 处理分页、ID、输入；`admin_routes.go` 注册五个管理 API。`dto/admin.go` 显式公开字段；`admin_repository.go` 执行参数化分页、真实全局聚合、锁和审计；`admin_service.go` 实施平台权限、状态与归档事务。`platform_audit.go` 与迁移 12 单独记录平台事件，不制造虚假团队归属。

用户状态变更按 ID 锁定操作者和目标，再验证操作者当前状态、角色与 session_version；状态实际变更递增目标 session_version 并写平台审计，提交后清理 Redis 全部会话。Redis 清理失败只记录不含凭据的警告，不撤销已提交状态；数据库鉴权仍拒绝旧会话，启用时再次递增版本，避免凭据复活。并发管理员交叉禁用由相同账号行锁串行化，后一个操作者被判失效，不能双双成功。

平台归档先锁团队，再锁并校验操作者，与普通业务“团队优先”的锁序一致；归档、团队日志、平台审计同事务。平台管理员不加入团队，不得到其项目/任务访问权。用户禁用不改变成员、owner 或任务分配，保持原始业务事实。

`AdminView.vue` 提供平台统计、用户管理和团队查看，操作必须填写原因并确认；路由守卫与后端分别拒绝普通用户。无权限或故障时清空旧列表并显示实际错误。E2E 脚本通过真实 CLI 初始化随机管理员，再经真实注册接口创建普通账号；私有临时凭据只在独立测试栈内使用并单文件清理，不降低生产限流。

## 阶段 8 交付工具

`cmd/checkdocs/main.go` 校验 OpenAPI、操作 ID、内部引用和本地 Markdown 链接；`internal/router/contract_test.go` 将发布契约与 Gin 实际注册路由逐项比对，防止文档与实现漂移。`scripts/init-env.mjs` 仅在 .env 不存在时以 0600 创建随机配置，保护已有数据连接凭据。`scripts/verify.sh` 统一执行编译、类型、真实集成和两类浏览器验证，失败即非零返回，不增加业务中间件。

`frontend/tests/release/workflow.spec.ts` 在连续流程中完成两个普通账号注册、团队邀请、项目、分配、拖拽、评论、日志、统计和管理员禁用；随后在独立栈暂停 MySQL、停止 Redis并验证恢复。`playwright.release.config.ts` 与模块回归分开运行，从全新临时数据库开始，保留实际限流且不影响演示主库。

最终 README、演示流程、测试矩阵与学习指南以实际实现为准，历史阶段文档保留各次范围与记录。业务迁移仍为 12，首版功能边界不变。
