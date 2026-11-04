# 阶段 5：评论与操作日志验收

> 此为历史阶段记录。阶段 6 已交付数据统计，当前运行和验收见 [阶段 6 验收](acceptance-stage-6.md)。

## 运行、迁移与停止

```sh
cd /Users/hj/Desktop/Go/TaskFlow
docker compose up --build -d
docker compose ps -a
curl -i http://localhost:8080/api/v1/health/ready
```

访问 http://localhost:8080。无需新增环境变量，沿用可信 .env。迁移 11 创建 comments 表，迁移服务应为 Exited (0)；健康检查 200，mysql/redis/migrations 都为 ok。既有账号、团队、项目、任务和日志保留，不修改旧迁移，不删除数据卷。

本地开发与 README 相同：shell 加载 .env，Compose 启动 mysql/redis；backend 先 go run ./cmd/migrate，再 go run ./cmd/server；frontend 运行 npm ci、npm run dev，通过 http://localhost:5173 的同源代理访问。

停止并保留业务数据：`docker compose down`。测试后可用 `docker compose --profile test stop mysql-test redis-test` 停止独立测试依赖。

## 实现内容与核心 Go 文件

- 评论分页、发表、作者编辑、作者/团队管理员删除、软删除、版本冲突、归档只读及当前成员校验。
- 评论嵌入任务详情和看板抽屉，真实加载/空数据/失败状态；纯文本展示，支持多行，不渲染 HTML。
- 团队与项目动态统一使用日志查询组件，增加独立 `/teams/:id/activities`、`/projects/:id/activities` 页面。
- 日志可组合筛选操作、目标类型、操作者、项目和时间，真实分页，历史离队成员的操作仍可查询。
- 评论变更与 comment.created/updated/deleted 日志同一 MySQL 事务；日志不复制评论正文。

核心文件：`comment_handler.go` 解析 HTTP 参数；`comment_service.go` 实施权限、版本与事务；`comment_repository.go` 负责分页、锁和持久化；`dto/comment.go` 公开安全响应；`model/comment.go` 映射数据库；`comment_routes.go` 注册登录与限流中间件。`activity_handler.go`、`activity_service.go` 统一解析和校验日志筛选，`activity_repository.go` 在团队/项目范围内执行参数化查询。完整路径与请求流见 [架构说明](architecture.md)。

## API 与规则

统一前缀 /api/v1，全部登录保护；ID 输出为字符串，审计时间 UTC。完整 schema 见 [OpenAPI 0.6.0](openapi.yaml)。

| 方法与路径 | 请求/响应 |
|---|---|
| GET /tasks/:taskId/comments | page/page_size，默认 1/20、最大 100；200 评论分页 |
| POST /tasks/:taskId/comments | content 必填；201 Comment |
| PATCH /comments/:commentId | content、version 必填；200 Comment |
| DELETE /comments/:commentId?version=N | version 必填；204，无响应体 |
| GET /teams/:teamId/activities | 分页加可选筛选；200 日志分页 |
| GET /projects/:projectId/activities | 分页加可选筛选；200 本项目日志分页 |

评论 content 去除首尾空白后为 1—2000 Unicode 字符；空白、无内容、未知 JSON 字段、无效版本返回 400。请求 JSON 最多 16 KiB。评论按 ID 倒序，软删除内容不返回。Comment 包括 id/task_id/author_id/author_nickname/content/version/can_edit/can_delete/created_at/updated_at。

评论独立 version 初始为 1，每次编辑或软删除递增。编辑与删除必须携带刚读取的版本，过期返回 409/COMMENT_VERSION_CONFLICT。页面取消旧编辑状态，刷新服务器内容并提示重新编辑，避免覆盖其他窗口的修改。评论操作不改变任务 version、位置或项目看板 revision。

当前团队成员可查看、发表；仅作者可编辑；作者、团队 owner/admin 可删除。团队管理员不能编辑他人的原文。平台 admin 不自动获得团队权限。归档团队或项目评论可读但不可发表/编辑/删除；任务删除、评论删除、非成员或离队用户不可读写对应评论，返回 404；同队越权为 403；限流 429；依赖/数据库故障 503。每用户每小时最多 120 次发表请求（无效发表请求也占次数）。

日志筛选：

| 参数 | 规则 |
|---|---|
| action | 精确代码，例如 comment.created/comment.updated/comment.deleted、task.moved；允许的真实代码见页面选项；未知类型 400 |
| target_type | team/user/invite/project/task/comment |
| actor_id | 正整数用户 ID，允许历史离队成员 |
| project_id | 可选正整数项目 ID；与 URL 所属团队/项目同时限制，不扩大权限 |
| from、to | 含时区的 RFC3339 时间，边界包含；开始晚于结束返回 400 |
| page、page_size | 默认 1/20，最多每页 100 条 |

页面日期选择按上海日历日发送：开始 00:00:00+08:00，结束 23:59:59.999999+08:00；服务端转换 UTC。日志默认 ID 倒序，团队范围包含其项目/任务/评论事件。历史记录不提供编辑、删除接口。关联资源已软删除时日志仍保留，点击详情会收到真实 404。

## 自动测试

```sh
cd /Users/hj/Desktop/Go/TaskFlow/backend
go test ./...
go vet ./...
go build ./cmd/server ./cmd/migrate
cd ..
./scripts/test-integration.sh
```

普通 go test 未设置 TASKFLOW_INTEGRATION 时明确跳过外部依赖测试；脚本连接独立 taskflow_test MySQL 与 Redis，执行真实 Gin API。新增覆盖权限与输入、软删除、项目/团队归档、任务删除、成员移除、版本冲突、并发编辑/移除、业务和日志回滚、23 条评论分页、日志组合筛选/时间边界/隔离、真实 Redis 限流。

```sh
cd /Users/hj/Desktop/Go/TaskFlow/frontend
npm ci
npm run build
npx playwright install chromium
cd ..
./scripts/test-e2e.sh
# 只运行本模块：
./scripts/test-e2e.sh comments.spec.ts
```

也可使用本机 Chrome：`PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' ./scripts/test-e2e.sh`。脚本在 18080 启动独立真实应用栈，测试后停止，不影响 8080 演示数据、会话或限流。E2E_WEB_PORT 可更改端口。不要用前端 mock 代替这些检查。

## 可执行 API 示例

先用自己的真实账号登录，替换邮箱/密码与任务 ID；命令会创建真实评论。Cookie 文件不要提交。

```sh
COOKIE=/tmp/taskflow-stage5.cookies
curl -fsS -c "$COOKIE" -H 'Content-Type: application/json' \
 -d '{"email":"替换为你的邮箱","password":"替换为你的密码"}' \
 http://localhost:8080/api/v1/auth/login
chmod 600 "$COOKIE"
TASK_ID=替换为真实任务ID
curl -fsS -b "$COOKIE" -H 'Content-Type: application/json' \
 -d '{"content":"接口联调完成，等待验收。"}' \
 "http://localhost:8080/api/v1/tasks/$TASK_ID/comments"
curl -fsS -b "$COOKIE" "http://localhost:8080/api/v1/tasks/$TASK_ID/comments"
```

使用返回的实际评论 ID/version 编辑，再次使用旧 version 应为 409：

```sh
COMMENT_ID=替换为刚创建的评论ID
COMMENT_VERSION=1
curl -i -b "$COOKIE" -X PATCH -H 'Content-Type: application/json' \
 -d "{\"content\":\"补充验收结论。\",\"version\":$COMMENT_VERSION}" \
 "http://localhost:8080/api/v1/comments/$COMMENT_ID"
```

更新成功后删除应传新的 version。查询真实评论日志：

```sh
PROJECT_ID=替换为真实项目ID
curl -fsS -b "$COOKIE" --get \
 --data-urlencode 'action=comment.created' --data-urlencode 'target_type=comment' \
 "http://localhost:8080/api/v1/projects/$PROJECT_ID/activities"
```

## 浏览器操作与明确验收标准

1. A 创建团队/项目/任务，B 用邀请码加入，C 不加入。B 在任务详情发表多行评论，刷新后内容保留；任务抽屉中 A 也能看到同一内容和作者。
2. 输入 `<img src=x onerror=alert(1)>` 等文本，页面显示原文，不生成图片或执行脚本；空白评论被阻止，后端直接请求也拒绝。
3. B 可编辑/删除自己的评论；A 可删除 B 的评论，但没有编辑入口；其他普通成员不能编辑或删除 B 的评论。直接 API 越权同样失败。
4. 同一作者两个窗口同时编辑：第一窗口保存成功，第二窗口收到冲突；旧编辑关闭、最新内容显示，不覆盖第一次结果。重新编辑后可正常保存。
5. 删除评论后刷新不再显示，数据库保留 deleted_at；动态出现对应删除事件，历史正文不通过日志再次暴露。
6. 23 条评论第一页 20 条、第二页 3 条；创建后回到第一页，删除某页最后一条后正确调整页码。空列表有明确状态。
7. 团队/项目“动态”可按操作、操作者、目标、项目和日期组合查询；完整日志页可独立刷新。未匹配时显示零记录，重置恢复实际记录，翻页数据正确。
8. 创建、编辑、移动任务及评论后，日志只出现在对应团队/项目；离队成员的旧操作仍保留。日志不应复制评论正文、密码或邀请码。
9. C 或已退出/被移除的 B 直接访问已知任务/日志 URL，返回无权限/404，不残留上一团队列表。
10. 归档项目或团队后刷新，评论继续可读、写入口消失；直接写 API 返回 409。任务软删除后，评论 API 不再可访问，日志仍可查。
11. 390px 下评论和日志筛选不造成整页横向溢出，长文本换行，按钮可操作；网络/数据库异常显示真实错误和重试，不显示伪造成功。
12. 自动回滚测试注入日志失败：创建、编辑、删除评论均整体回滚，版本/正文/删除标记和日志数不发生部分提交。

## 限制与下一模块

首版纯文本、倒序分页，不含 Markdown、回复楼层、@ 提及、附件或实时推送。评论会刷新当前页；其他窗口需手动刷新或在冲突后读取最新内容。软删除/归档不提供恢复接口。

统计和管理员后台尚未实现。下一模块为阶段 6：按状态、优先级、负责人及逾期汇总真实任务并展示统计图表。
