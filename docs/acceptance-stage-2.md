# 阶段 2：团队管理验收

本页保留阶段 2 的交付边界；项目现已实现，当前验收见 [阶段 3](acceptance-stage-3.md)。

## 运行与升级

```sh
cd /Users/hj/Desktop/Go/TaskFlow
docker compose up --build -d
docker compose ps -a
```

访问 http://localhost:8080，注册或使用已有账号登录，然后进入团队空间。阶段 1 的账号和数据卷保留；迁移服务执行版本 3 至 6，创建 teams、team_members、team_invites、activity_logs。migrate 为 Exited (0) 属于正常状态。

本阶段不增加必填环境变量。MYSQL_*、REDIS_ADDR、APP_ORIGIN 及本地 Go/Vite 启动方式沿用 README。停止并保留演示数据：docker compose down。不要重新覆盖已有 .env 密码。

## 已实现接口

所有接口均需要登录，统一前缀 /api/v1。列表返回 items/page/page_size/total，默认 20 条、最多 100 条；ID 为字符串。完整请求和响应见 openapi.yaml。

| 方法与路径 | 作用与返回 |
|---|---|
| GET /teams | 本人当前加入的团队，支持 status=all/active/archived，默认 all |
| POST /teams | name（1—80 字符）、description（最多 2000 字符）；201 团队 DTO |
| GET /teams/:teamId | 团队详情，含 my_role、member_count、archived_at |
| PATCH /teams/:teamId | 更新 name/description，至少一项；200 团队 DTO |
| POST /teams/:teamId/archive | 所有者归档；204 |
| GET /teams/:teamId/members | 当前成员、角色、账号状态和加入时间 |
| PATCH /teams/:teamId/members/:userId | 所有者设置 role=admin/member；204 |
| DELETE /teams/:teamId/members/:userId | 移除成员；204 |
| POST /teams/:teamId/leave | 退出团队；204 |
| POST /teams/:teamId/transfer-ownership | user_id 字符串，原所有者成为管理员；200 团队 DTO |
| POST /teams/:teamId/invites | expires_in_days 默认 7（1—30）、max_uses 默认 20（1—100）；201 invite/code |
| GET /teams/:teamId/invites | 邀请元数据，不返回原始码或摘要 |
| DELETE /teams/:teamId/invites/:inviteId | 撤销邀请码，可重复调用；204 |
| POST /teams/join | code；200 team/already_joined |
| GET /teams/:teamId/activities | 当前成员可查看真实团队动态 |

团队外用户、已退出或被移除的用户请求团队资源返回 404。团队内角色不满足返回 403；已归档团队写入返回 409；非法邀请码返回 400；依赖故障返回 503。平台管理员没有隐式团队权限。

## 三账号浏览器验收

准备 A、B、C 三个独立浏览器窗口（或不同浏览器上下文），分别注册登录。

1. A 创建团队，查看详情和成员页；A 为所有者，成员数为 1。团队动态能看到创建记录。
2. A 在邀请码页创建邀请码，默认有效期 7 天、20 次，复制完整 64 位码。刷新页面后原始码不再显示，但元数据仍在。
3. B 使用邀请码加入，为普通成员。A 刷新后看到 B，邀请码已使用次数增加 1。B 重复使用同一邀请码不会重复入队或扣次数。
4. C 未加入时，即使知道详情 URL 也不能查看团队、成员和动态。
5. B 不能编辑团队、管理邀请码、设置角色、转让或归档。A 将 B 设为管理员后，B 刷新可编辑团队及创建/撤销邀请码，但仍不能任命管理员、移除其他管理员、转让或归档。
6. A 在成员页将所有权转给 B，确认后 A 变为管理员、B 为唯一所有者。A 退出团队后立即失去访问权限。
7. B 新建邀请码并撤销，C 使用它加入被拒绝。耗尽或过期的邀请码同样不能让新成员加入。
8. B 归档团队，仍能查看资料、成员、邀请元数据和动态，但全部写操作隐藏或被服务端拒绝。C 即使持有有效邀请码也不能加入。
9. 团队列表的“全部状态/进行中/已归档”筛选正确；刷新和重新登录后数据保持。
10. 390px 窄屏团队列表无横向溢出；成员表在表格区域内横向滚动，不撑开整个页面。

补充权限验收：管理员只能移除普通成员，不能移除所有者或其他管理员；所有者不能直接退出或修改自己的角色，必须先转让；所有者可移除其他管理员。移除或退出采用成员软移除，不物理删除历史记录，重新加入后恢复为普通成员。

## 并发、一致性与日志

- 所有团队写操作在同一 MySQL 事务中先锁团队，再读取当前成员角色。
- 单次邀请码被两名新用户并发使用：只能一人成功，used_count 不超过 max_uses。
- 同一用户并发重复加入：只有一个活跃成员关系、一次扣次数和一条 member.joined 日志。
- 两个所有权转让同时提交：只能一个成功，teams.owner_id 与唯一活跃 owner 一致。
- 业务写入、邀请码计数和对应 activity_logs 同事务；日志写失败时全部回滚。
- 邀请码原始值和摘要不进入活动日志；列表只返回邀请元数据。
- 已加入用户重试同一个已知邀请码，即使码后来过期、撤销或耗尽，也返回 already_joined=true，不增加计数和日志；归档团队始终拒绝加入。

当前动态页是基础审计查看，支持分页。阶段 5 再增加完整日志检索和项目日志能力。

## 自动测试

```sh
cd /Users/hj/Desktop/Go/TaskFlow/backend
go test ./...
go vet ./...
go build ./cmd/server ./cmd/migrate
cd ..
./scripts/test-integration.sh
```

集成测试使用独立的 taskflow_test 数据库和 Redis 测试服务，不操作日常演示数据库。覆盖团队生命周期、跨团队隔离、完整角色边界、邀请摘要、撤销/过期/耗尽、重复加入、重新加入、所有权并发、邀请码并发、归档只读、分页默认值、日志与业务回滚。日志故障由测试 DB 的 GORM 回调注入，不改变生产接口。

前端构建与真实端到端测试：

```sh
cd /Users/hj/Desktop/Go/TaskFlow/frontend
npm ci
npm run build
npx playwright install chromium
npm run test:e2e
```

也可使用本机 Chrome：

```sh
PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' npm run test:e2e
```

需要目标应用已经启动；TASKFLOW_E2E_URL 可指定地址。测试创建真实随机邮箱账号及团队，完成后保留记录；团队测试最终归档其团队，不伪造 API 响应。输出按运行生成唯一目录，不批量清理已有文件。

## 当前边界与下一模块

- 创建团队：每用户每小时 20 次；创建邀请码：每用户每小时 30 次；加入团队：每用户每分钟 30 次。
- 邀请原始码仅创建响应返回；当前页面可复制，丢失后需要新建并可撤销旧码。
- 归档后不可编辑、邀请、加入、变更角色、退出或转让；首版不提供恢复归档。
- 所有权只能转给团队内账号状态正常的其他成员。
- 项目、任务尚未实现，没有假项目列表或任务数。阶段 4 引入任务后，必须在当前移除/退出事务里同步清空任务负责人；本阶段没有任务表可更新。
- Redis/数据库故障时保留错误与重试入口，不以空列表冒充成功。
- 下一模块为阶段 3：项目管理。
