# 阶段 6：数据统计验收

> 此为历史阶段记录。阶段 7 已交付管理员后台，当前运行和验收见 [阶段 7 验收](acceptance-stage-7.md)。

## 启动、升级与停止

```sh
cd /Users/hj/Desktop/Go/TaskFlow
docker compose up --build -d
docker compose ps -a
curl -i http://localhost:8080/api/v1/health/ready
```

访问 http://localhost:8080。团队详情、项目详情及项目看板有“数据统计”链接；也可访问 /teams/:teamId/statistics 或 /projects/:projectId/statistics。

无新增环境变量、数据库迁移或前端依赖，迁移版本仍为 11。migrate 为 Exited (0)，就绪检查应为 200，三个 checks 均为 ok。原账号、团队、项目、任务、评论和日志保留，不删除数据卷。

本地开发：按 README 加载可信 .env，Compose 启动 mysql/redis；backend 先 go run ./cmd/migrate，再 go run ./cmd/server；frontend 执行 npm ci、npm run dev，访问 http://localhost:5173，使用同源 API 代理。

停止并保留业务数据：`docker compose down`。独立测试依赖可用 `docker compose --profile test stop mysql-test redis-test` 停止。

## 实现内容与核心 Go 文件

团队/项目统计支持任务总数、已完成、未完成、逾期、完成率，以及状态、优先级和负责人分布；页面以真实数字、条形图、环形图和负责人表呈现。空项目显示完整零值，加载/故障/无权限都有真实状态。

- `internal/dto/statistics.go`：明确响应字段、分组、ID 字符串、时区和查询时间。
- `internal/repository/statistics_repository.go`：MySQL COUNT/SUM/GROUP BY 聚合，不读取前端已加载的任务页，不使用缓存。
- `internal/service/statistics_service.go`：当前成员权限、同一读事务、日期口径、完成率、零值补全。
- `internal/handler/statistics_handler.go`：解析资源 ID，调用服务，统一响应。
- `internal/router/statistics_routes.go`：登录保护和禁止缓存的两个 GET 路由。
- `tests/integration/statistics_test.go`：真实权限、数据变化、全量聚合、快照和异常测试。

前端 `views/statistics/StatisticsView.vue` 复用团队/项目页面，`api/statistics.ts` 访问真实接口，图表直接使用响应数据。完整请求流及事务说明见 [架构说明](architecture.md)。

## API 与口径

统一前缀 /api/v1。全部登录保护，仅当前团队成员可访问。普通成员可查看；平台 admin 没有隐式团队权限。非成员、离队或被移除用户、不存在的资源返回 404；未登录 401，无效 ID 400，依赖/数据库异常 503。

| 方法与路径 | 响应 |
|---|---|
| GET /teams/:teamId/statistics | 200，团队所有项目的真实任务统计，project_id=null |
| GET /projects/:projectId/statistics | 200，当前项目的真实任务统计 |

成功仍为 data/request_id；详细结构见 [OpenAPI 0.7.0](openapi.yaml)。不支持日期范围、状态筛选或趋势查询；统计全部符合范围的未删除任务。

| 字段 | 含义 |
|---|---|
| scope、team_id、project_id、name | 当前统计范围和名称，ID 为字符串 |
| archived | 团队或当前项目已归档，仍可查看 |
| as_of、today、timezone | UTC 查询时间、上海当天日期、Asia/Shanghai |
| total | 全部未软删除任务，包含已完成及归档项目任务 |
| completed | status=done |
| uncompleted | total-completed |
| overdue | status 非 done，且 due_at 早于 today |
| completion_rate | completed/total×100，四舍五入到一位小数；total=0 时为 0 |
| by_status | 固定 todo/in_progress/in_review/done 四组，缺失补零 |
| by_priority | 固定 low/medium/high/urgent 四组，缺失补零 |
| by_assignee | user_id、nickname、total、completed、overdue；只包含有任务的负责人及始终存在的未分配组 |

今日截止、无截止日期、已完成任务不算逾期。各图百分比保留一位小数，独立取整后相加可能略偏离 100%，数量始终精确。

负责人 user_id=null 表示未分配，不代表匿名用户；未分配没有任务时也返回零组。其他无任务成员不返回，不能用分组数当团队人数。成员离队/被移除后的清理完成后，其任务自然汇总到未分配组。

归档不删除任务，也不改变完成状态，因此归档前后统计通常不变；软删除排除计数。团队统计包含其归档项目，项目统计检查父团队权限。评论操作不改变这些任务计数。

全部聚合与权限读取在同一 MySQL 默认 REPEATABLE READ 事务中，避免并发修改使总数与分组相互矛盾。数据库故障返回错误，不能显示成零值；后续请求能看到新提交任务。

## 可执行 API 与自动测试

使用已有真实账号登录。替换邮箱、密码及 ID；Cookie 文件不要提交。

```sh
COOKIE=/tmp/taskflow-stage6.cookies
curl -fsS -c "$COOKIE" -H 'Content-Type: application/json' \
 -d '{"email":"替换为你的邮箱","password":"替换为你的密码"}' \
 http://localhost:8080/api/v1/auth/login
chmod 600 "$COOKIE"
TEAM_ID=替换为真实团队ID
PROJECT_ID=替换为真实项目ID
curl -fsS -b "$COOKIE" "http://localhost:8080/api/v1/teams/$TEAM_ID/statistics"
curl -fsS -b "$COOKIE" "http://localhost:8080/api/v1/projects/$PROJECT_ID/statistics"
```

真实集成测试：

```sh
cd /Users/hj/Desktop/Go/TaskFlow/backend
go test ./...
go vet ./...
go build ./cmd/server ./cmd/migrate
cd ..
./scripts/test-integration.sh
```

普通 go test 未开启 TASKFLOW_INTEGRATION 时明确跳过外部依赖；脚本连接独立 taskflow_test 与 Redis。新增检查：零值、日期边界、按状态/优先级/负责人汇总、任务编辑/移动/删除后的变化、归档与离队、跨团队权限、105 条任务不受分页限制、聚合中间发生实际写入仍保持快照、数据库失败不泄露 SQL。

前端和浏览器：

```sh
cd /Users/hj/Desktop/Go/TaskFlow/frontend
npm ci
npm run build
npx playwright install chromium
cd ..
./scripts/test-e2e.sh
# 仅检查统计模块：
./scripts/test-e2e.sh statistics.spec.ts
```

可用本机 Chrome：`PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' ./scripts/test-e2e.sh`。脚本使用 18080 的独立真实应用与 MySQL/Redis，测试后停止，不改变主库或会话；E2E_WEB_PORT 可调整端口。

## 浏览器验收标准

1. A 建团队，B 加入，C 不加入。A/B 均能打开团队和项目统计；C 或退出后的 B 即使知道 URL 也收到 404；直接 API 同样拒绝。
2. 空项目的总数、完成、未完成、逾期、完成率均为 0，四组状态与优先级完整，无 NaN、虚构趋势或占位任务。
3. 创建四项真实任务，各用一种状态/优先级。刷新后总数为 4，各状态与优先级计数为 1，完成 1、未完成 3、完成率 25%。
4. 设置未完成任务昨天截止应逾期，今天截止/无日期不逾期；过去截止但已完成也不逾期。页面显示本次采用的上海日期。
5. 将逾期任务移到已完成，刷新统计后完成数增加、逾期减少；删除任务后总数减少。3 项中 1 项已完成显示 33.3%。
6. 修改负责人和优先级，刷新后分组准确变化；移除成员/成员离队后，原分配进入未分配组，其他团队计数不变。
7. 团队有多个项目时汇总全部项目。归档项目仍计入；归档团队后仍能读取保留任务，页面明确归档状态。
8. 负责人行的总数之和等于 total，完成/逾期之和分别等于总体；状态、优先级各组数量之和也等于 total。
9. 超过 100 条任务仍全部计入，不受看板首批 50 或列表每页 20/100 限制。
10. 两个窗口更新任务后，另一个窗口点击刷新能得到实际新统计；查询失败显示错误并清空旧统计，不把失败渲染成全零成功。
11. 390px 下卡片、条形图、环形图可读，整页无横向溢出，负责人表在局部滚动；统计请求无修改任务/日志的副作用。

## 限制与下一模块

首版为当前状态快照，无历史趋势、导出、日期筛选或实时推送。负责人聚合一次全部返回，适合中小团队；超大团队后续增加分页。MySQL 保持默认 REPEATABLE READ，若更改隔离级别需同步维护快照保证。

管理员后台仍未实现。下一阶段 7：安全初始化管理员、平台统计、用户禁用并撤销会话、团队查看与归档。
