# 阶段 3：项目管理验收

> 此为历史阶段记录。当前阶段 4 已交付任务与看板，最新运行和验收见 [阶段 4 验收](acceptance-stage-4.md)。

## 运行与升级

```sh
cd /Users/hj/Desktop/Go/TaskFlow
docker compose up --build -d
docker compose ps -a
```

访问 http://localhost:8080，登录后进入团队，在团队页点击“项目”。无新增必填环境变量。阶段 0—2 的账号、团队、会话及数据卷保留，迁移版本从 6 升至 8；migrate 为 Exited (0) 正常。

本地 Go/Vite：加载可信 .env，启动 MySQL/Redis，先 go run ./cmd/migrate，再 go run ./cmd/server 和前端 npm run dev。配置和端口沿用 README。

停止并保留演示数据：docker compose down。

## 本模块 API

统一前缀 /api/v1，全部要求登录。ID 为字符串，列表默认 20 条、最多 100 条，包含 items/page/page_size/total。完整结构见 openapi.yaml。

| 方法与路径 | 请求及成功响应 |
|---|---|
| GET /teams/:teamId/projects | page、page_size、status=all/active/archived（默认 all）；200 项目分页 |
| POST /teams/:teamId/projects | name（1—100 字符）、description（最多 2000 字符，可省略）；201 项目 DTO |
| GET /projects/:projectId | 200 项目 DTO |
| PATCH /projects/:projectId | name/description 至少一项；200 项目 DTO |
| POST /projects/:projectId/archive | 无请求体；204 |
| GET /projects/:projectId/activities | 分页查询项目动态；200 日志分页 |

项目 DTO 包含 id、team_id、team_name、my_role、team_archived_at、name、description、created_by、creator_nickname、archived_at、created_at、updated_at。创建者与所属团队不可修改，未知字段被拒绝；不提供项目物理删除和恢复归档接口。

日志响应新增 project_id：团队级操作为 null，项目操作为字符串 ID。同一项目事件同步出现在团队动态和本项目动态。

## 权限与状态

- 团队所有当前成员均可查看项目和项目动态，无独立项目成员表。
- 只有当前团队 owner/admin 可以创建、编辑和归档。
- 创建者没有额外权限：降为 member 后只读，退出或被移除后立即失去项目访问权。
- 平台管理员没有隐式团队或项目访问权。
- 非成员、跨团队和不存在的项目统一 404；普通成员尝试管理项目返回 403。
- 归档项目仍可读；管理员编辑或重复归档返回 409。归档整个团队后，其下所有项目均只读，创建、编辑、归档项目均被拒绝。
- status 仅按项目自身 archived_at 筛选。团队归档不会改写每个项目的 archived_at，页面以 team_archived_at 显示团队只读状态。
- 项目名称前后空白移除，允许同队同名，用 ID 区分。
- 400 参数错误；401 登录失效；403 权限不足；404 不可访问；409 状态/并发冲突；429 限流；503 依赖故障。项目创建每用户每小时最多 60 次。

## 多账号手工验收

1. A 创建团队，B 使用邀请码加入。C 保持未加入。
2. A 在团队页点击“项目”，创建项目，确认详情页内容来自实际 API；刷新后仍存在。
3. A 修改项目名称/简介，保存并刷新；项目动态出现创建和修改记录，团队动态也出现同一项目事件并可跳转查看项目。
4. B 能查看列表、详情及动态，但没有创建、保存和归档按钮。直接请求管理 API 也返回 403。
5. C 知道项目 ID 仍不能查看详情、列表或日志；404 不暴露其他团队的数据。
6. A 将 B 升为 admin，B 刷新后可编辑项目；再降回 member，后端立即拒绝修改。移除或退出后，项目创建者也不能继续访问。
7. A 归档项目，内容和历史保留，编辑入口消失；直接请求编辑、再次归档返回 409。列表“已归档”可找到该项目。
8. A 创建另一个未归档项目，再归档团队；该项目也进入团队只读状态，不能再创建或修改项目。
9. 在 390px 窄屏查看项目列表与详情，无整页横向溢出；加载、空列表、无权限和服务故障有真实提示。

## 自动检查

```sh
cd /Users/hj/Desktop/Go/TaskFlow/backend
go test ./...
go vet ./...
go build ./cmd/server ./cmd/migrate
cd ..
./scripts/test-integration.sh
```

真实集成测试使用独立 taskflow_test 数据库和 Redis。新增覆盖：项目生命周期、跨团队/平台角色隔离、成员权限即时变化、创建者无特权、部分更新、非法字段、项目/父团队归档、日志范围、并发移除/编辑、并发归档/编辑、业务与日志回滚、分页、组合外键以及迁移中断恢复。

前端：

```sh
cd /Users/hj/Desktop/Go/TaskFlow/frontend
npm ci
npm run build
npx playwright install chromium
cd ..
./scripts/test-e2e.sh
```

使用本机 Chrome 可跳过下载：

```sh
PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' ./scripts/test-e2e.sh
```

独立 compose.e2e.yaml 在 18080 启动真实应用，使用独立网络、MySQL 和 Redis，不共享 8080 演示数据库、会话或限流计数。E2E_WEB_PORT 可更改测试端口。脚本等待健康后执行测试，并在退出时停止测试服务；测试 MySQL 使用 tmpfs，Redis 不持久化，测试状态随测试服务停止消失，不批量删除工作区文件。

npm run test:e2e 仍可直接针对已运行的目标；用 TASKFLOW_E2E_URL 指定地址。直接以 8080 为目标会创建真实测试账号与记录，日常回归使用独立脚本。

## 数据与事务验收

- 项目写入固定锁序：团队 → 项目；在锁内重新查询成员角色并检查父团队和项目状态。
- 移除成员或归档与项目修改并发时，修改只能在权限/状态改变前完成，否则被拒绝；不会出现归档后成功写入的日志。
- 创建、修改和归档与 activity_logs 同事务；日志写入失败时项目变更全部回滚。
- projects 对团队和创建者有外键；activity_logs 的 (team_id,project_id) 组合外键拒绝跨团队关联。
- 迁移 8 通过精确检查外键列关联，支持 DDL 已成功、迁移版本尚未记录时安全重试；测试仅在独立测试库模拟该中断。

## 交付边界

当前实现项目基础管理和真实动态。任务看板、任务分配、任务数量、评论、统计均未实现，页面不展示模拟任务或伪造计数。下一模块为阶段 4：任务与看板。
