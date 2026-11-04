# 阶段 4：任务与看板验收

> 此为历史阶段记录。阶段 5 已交付评论与日志查询，当前运行和验收见 [阶段 5 验收](acceptance-stage-5.md)。

## 运行、迁移和停止

```sh
cd /Users/hj/Desktop/Go/TaskFlow
docker compose up --build -d
docker compose ps -a
curl -i http://localhost:8080/api/v1/health/ready
```

访问 http://localhost:8080，登录 → 团队 → 项目 → 项目详情的“任务看板”。左侧“我的任务”查看个人分配。没有新增必填环境变量，沿用可信 .env。迁移 9 新增 tasks，迁移 10 新增 projects.task_revision；不改写旧迁移、不删除账号、团队、项目或数据卷。migrate 显示 Exited (0)，就绪检查应为 200，三个 checks 都为 ok。

本地开发沿用 README：加载 .env，Compose 启动 mysql/redis；backend 中先 go run ./cmd/migrate，再 go run ./cmd/server；frontend 中 npm ci、npm run dev。Vite 访问 http://localhost:5173，同源代理 API；后端 8081 不直接跨域访问。

停止并保留数据：`docker compose down`。仅停止测试依赖：`docker compose --profile test stop mysql-test redis-test`。测试和演示环境互不共享数据库或 Redis。

## 本模块内容和核心文件

任务增删改、单负责人、四种优先级、截止日期、四列看板、拖拽/同列排序、列表筛选、我的任务、乐观锁、看板分页快照、离队清理均已连接真实后端。任务操作写入已有团队/项目动态，事务失败不会留下成功日志。

Go 核心职责：task_handler.go 解析 HTTP 参数；task_service.go 处理成员权限、业务规则和事务；task_repository.go 执行查询、重排和分配清理；dto/task.go 显式公开响应并区分空值与未修改；model/task.go 映射表；task_routes.go 注册登录保护的接口。完整路径和请求流见 [架构说明](architecture.md)。

## API

统一前缀 /api/v1，全部要求登录。ID 是字符串，时间 UTC，due_at 是不含时区的 YYYY-MM-DD。创建 201、读取/编辑/移动 200、删除 204。未知 JSON 字段拒绝；JSON 请求体最多 16 KiB。

| 方法与路径 | 参数和响应 |
|---|---|
| GET /projects/:projectId/tasks | 分页及筛选；items/page/page_size/total/revision |
| POST /projects/:projectId/tasks | title 必填，description/priority/assignee_id/due_at 可选；Task |
| GET /projects/:projectId/board | 筛选；project、revision、四个 columns，每列 items/total/page_size/has_more |
| GET /tasks/:taskId | Task，附带当前角色、归档状态、can_delete、is_overdue、version |
| PATCH /tasks/:taskId | version 必填，至少一个资料字段；Task |
| POST /tasks/:taskId/move | version、status 必填，before_task_id 可空；Task |
| DELETE /tasks/:taskId?version=N | version 查询参数必填，软删除并调整原列顺序 |
| GET /me/tasks | 分页及筛选，items/page/page_size/total；仅限分配给当前用户且仍可访问的任务 |

字段规则：title 去前后空白后 1—200 字符，description 最多 2000 字符；status 为 todo/in_progress/in_review/done，创建固定 todo；priority 为 low/medium/high/urgent，默认 medium。负责人必须是当前活跃团队成员；null 表示未分配。due_at 可空，合法年份 1000—9999。

PATCH 省略字段保持原值，assignee_id/due_at 显式 null 真正清空，description 传空字符串可清空。状态/position 只能通过 move 调整，禁止塞入 PATCH。move 的 before_task_id 必须是同项目目标列中另一任务，null 或省略放列末；不能以自身作目标。

筛选支持 keyword（标题/描述，最多 100 字符，百分号和下划线按字面匹配）、status、priority、assignee_id（用户 ID 或 unassigned）、overdue=true/false。我的任务不支持 assignee_id/revision。列表默认 page=1、page_size=20，最多 100 条。项目列表按列顺序和 position/id 排序；我的任务按有日期优先、日期升序、更新时间和 ID 倒序。

看板每列首批 50 条；更多请求示例为 GET /projects/:id/tasks?status=todo&page=2&page_size=50&revision=N。看板改变后旧 revision 返回 409/BOARD_CHANGED，页面自动重新加载，不拼接新旧快照。筛选会改变真实计数，没有结果的列为零。

逾期按 Asia/Shanghai 日历日：未完成且 due_at 早于当天；今日截止、无日期、已完成都不算逾期。

## 可执行 API 与回归测试

```sh
cd /Users/hj/Desktop/Go/TaskFlow/backend
go test ./...
go vet ./...
go build ./cmd/server ./cmd/migrate
cd ..
./scripts/test-integration.sh
```

task_test.go 直接通过 Gin 请求真实 MySQL/Redis，覆盖创建、部分更新、null 清空、删除、筛选、非法参数、当前成员权限、跨团队分配、平台角色无特权、归档、排序、过期版本、并发更新/移除、跨团队清理隔离、事务回滚、55 条分页快照和迁移中断恢复。未设置 TASKFLOW_INTEGRATION 时，普通 go test 明确跳过外部依赖测试；完整运行使用上述脚本。

已有账号也可手工调用。替换邮箱、密码、项目 ID 为自己的真实值；下面的操作会创建真实记录。Cookie 文件不要提交。

```sh
COOKIE=/tmp/taskflow-stage4.cookies
curl -fsS -c "$COOKIE" -H 'Content-Type: application/json' \
  -d '{"email":"替换为你的邮箱","password":"替换为你的密码"}' \
  http://localhost:8080/api/v1/auth/login
chmod 600 "$COOKIE"
PROJECT_ID=替换为已有项目ID
curl -fsS -b "$COOKIE" -H 'Content-Type: application/json' \
  -d '{"title":"联调任务接口","priority":"high","assignee_id":null,"due_at":null}' \
  "http://localhost:8080/api/v1/projects/$PROJECT_ID/tasks"
curl -fsS -b "$COOKIE" "http://localhost:8080/api/v1/projects/$PROJECT_ID/board"
curl -fsS -b "$COOKIE" http://localhost:8080/api/v1/me/tasks
```

使用响应中的实际 id/version 编辑或移动；重复使用旧 version 应返回 409。完整请求结构见 [OpenAPI](openapi.yaml)。

```sh
cd /Users/hj/Desktop/Go/TaskFlow/frontend
npm ci
npm run build
npx playwright install chromium
cd ..
./scripts/test-e2e.sh
```

已有 Chrome 时可用 `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' ./scripts/test-e2e.sh`。脚本在 18080 启动独立真实应用、等待就绪、运行 Playwright 并停止测试栈；E2E_WEB_PORT 可调整端口。直接 npm run test:e2e 默认为 8080，会在目标创建真实记录，日常使用独立脚本。

## 浏览器验收步骤与标准

1. A 创建团队和项目，B 加入团队，C 不加入。A/B 都能创建、编辑和移动任务；B 不需要团队 admin 角色。C 直接访问已知任务 ID 返回 404。
2. A 创建任务，填写标题/描述、选优先级和 B、设置日期。刷新后保留，卡片显示真实负责人和优先级。新任务在待办末尾。
3. 拖任务到进行中列，刷新后位置不变；拖到另一张卡上方应插到它前面。同列前移/后移按钮也能改变顺序；列数与分页计数真实更新。
4. 打开任务抽屉，编辑并保存；清空日期、负责人后保存再刷新仍为空。空标题、无效日期、非本团队负责人被拒绝。
5. B 打开“我的任务”，只看到分配给自己的任务，能筛选、分页和打开详情；日期早于今天且未完成显示逾期。移动到已完成后不再逾期。
6. 按关键词、状态、优先级、负责人、逾期组合筛选；无结果显示真实空状态。看板每列最多先显示 50 条，55 条场景加载更多得到 55 个不同 ID。
7. 两个浏览器同时打开任务，B 先保存，A 再保存旧版本：提示任务已被修改、重新加载服务器内容，不覆盖 B 的结果；A 重新编辑后可保存。
8. A 打开旧的 50 条快照，B 修改任务，A 加载更多：提示看板已变化并刷新；再加载获得全部 55 条，不重复或遗漏。拖拽失败同样刷新并提示。
9. B 不能删除 A 创建的任务，直接 DELETE 也为 403；B 可删除自己创建的任务；A/团队 admin 可删除其他任务。删除后列表和看板不再出现，数据库保留 deleted_at 历史，GET 返回 404。
10. A 移除 B，或 B 自行退出：该团队所有未删除任务的负责人立即清空，包括已完成和归档项目任务，版本/revision 同步变化，B 不再能访问这些任务；其他团队的分配不变。
11. A 归档项目或团队，仍可查看任务，没有创建/保存/移动/删除入口；直接写 API 返回 409，不新增成功日志。
12. 团队/项目动态出现 task.created/updated/moved/deleted/unassigned 真实事件；数据库/日志写入失败时任务、顺序、版本、分配清理、成员移除整体回滚。
13. 390px 手机查看：整页无横向溢出，看板内部可横向滚动；可切换列表，在详情选择状态并移到列末。加载、空列表、故障和无权限均有实际提示。

## 限制和下一模块

没有实时推送，其他用户更新后使用刷新或冲突提示获取最新数据；拖拽依赖桌面原生 API，移动端使用列表/详情和按钮。首版每次重排遍历相关列，适合中小项目；特别大的看板需要后续排序优化。没有恢复软删除或归档接口。

评论、统计、管理后台未实现，不显示假数据。下一阶段 5 实现评论增删改权限，并完善操作日志查询体验。已记录的团队、项目、任务事件会继续沿用。
