# 数据库设计

## 已实现

`schema_migrations(version BIGINT UNSIGNED PRIMARY KEY, name VARCHAR(255), applied_at DATETIME(6))`。
当前版本为 12。阶段 1 创建 users：id、email、nickname、password_hash、platform_role、status、session_version、created_at、updated_at。email 使用 ASCII 不区分大小写的唯一索引；昵称上限 40 字符；role/status 有 CHECK 约束。数据库不预置管理员或演示账号。

session_version 默认 1，改密时原子递增，用于撤销旧会话。Redis 仅保存会话摘要键、用户会话索引及带 TTL 的限流计数。阶段 2 已新增团队相关表，阶段 3 已创建 projects 并关联项目日志；阶段 4 已创建 tasks 并新增 projects.task_revision；阶段 5 已创建 comments。

## 整体数据关系（全部核心业务表已落地）

| 表 | 核心字段与约束 |
|---|---|
| users | email 唯一、nickname、password_hash、platform_role、status |
| teams | name、description、owner_id → users、archived_at |
| team_members | team_id → teams、user_id → users、role；(team_id,user_id) 唯一 |
| team_invites | team_id、token_hash 唯一、expires_at、max_uses、used_count、revoked_at、created_by |
| projects | team_id → teams、name、description、created_by、archived_at、task_revision |
| tasks | project_id → projects、title、description、status、priority、assignee_id 可空、created_by、due_at DATE、position、version、deleted_at |
| comments | task_id → tasks、author_id → users、content、version、deleted_at |
| activity_logs | team_id、project_id 可空、actor_id、action、target_type、target_id、metadata JSON、created_at |

业务实体使用 BIGINT 主键和 UTC created_at/updated_at。建立外键和查询索引，禁止级联物理删除业务历史。团队成员共享团队内全部项目，不另外引入项目成员表。

## 必须维持的规则

- 每队恰有一个 owner；owner_id 和成员角色在事务内保持一致。
- 负责人必须是当前团队成员；成员移除时同事务清空其任务分配。
- 邀请码仅保存摘要，校验有效期、撤销和使用次数；重复加入不扣次数。
- 任务状态 todo/in_progress/in_review/done；优先级 low/medium/high/urgent。
- 状态或内容修改校验 version；移动看板锁定项目后更新位置。
- 软删除任务不参与统计；团队和项目归档后仅可读。
- 平台管理员不隐式获得团队编辑权。

## 阶段 2 已落地的约束

- teams.owner_id → users.id；团队归档使用 archived_at。
- team_members 的 team_id/user_id 分别有外键；(team_id,user_id) 唯一，removed_at 为软移除，joined_at 记录最近加入时间。
- owner_guard 是生成列，只有活跃 owner 时为 1，其他情况为 NULL；(team_id,owner_guard) 唯一，限制一个活跃 owner。服务层同事务保证 owner_id 匹配且不会留下无 owner 团队。
- team_invites 仅保存 SHA256 token_hash；摘要唯一，有效期/撤销/最大次数参与校验；CHECK 阻止 used_count 超限。创建者和团队均有外键。
- activity_logs 的团队/操作者有外键，metadata 为 JSON，包含业务变化但不包含邀请码；team/id、project/id、team/created_at 建有索引。
- 团队事件的 activity_logs.project_id 为 NULL；阶段 3 的项目事件填入项目 ID 并由组合外键约束。
- 业务表不设级联物理删除；成员移除不删除活动历史。

## 阶段 3 已落地的约束

- projects.team_id → teams.id；projects.created_by → users.id。创建者离队不会丢失项目或更改创建记录，但失去访问权限。
- (team_id,id) 为唯一索引，(team_id,archived_at,id) 支持团队内筛选与分页。
- activity_logs(team_id,project_id) → projects(team_id,id)，同时校验项目存在且属于日志团队。
- 项目 name 最多 100 字符，description 最多 2000 字符；项目同名可共存，不提供更换团队或创建者接口。
- 项目归档采用 archived_at；父团队归档不会改变子项目 archived_at，但所有写操作均检查父状态。

## 阶段 4 已落地的约束与索引

- tasks.project_id → projects.id；created_by、可空 assignee_id → users.id。负责人必须是当前活跃团队成员，由团队锁内的 Service 校验，不能只依赖 users 外键。
- title 为 VARCHAR(200)，description 为 VARCHAR(2000)；status/priority 有 CHECK 约束；version 为正数，初始 1。
- due_at 使用可空 DATE；保存 YYYY-MM-DD，不做 UTC 日期偏移。逾期条件为“非 done 且日期早于上海当天”。
- position 使用无符号 BIGINT，从 0 开始；每列按 position/id 排序。重排在项目锁下保持连续，涉及行同步增加 version；不设置会妨碍中间交换的唯一位置索引。
- deleted_at 为软删除标记，列表、看板、我的任务及后续统计均排除已删除任务。
- idx_tasks_board(project_id,status,deleted_at,position,id)、idx_tasks_assignee(assignee_id,deleted_at,due_at,id)、idx_tasks_priority(project_id,priority,deleted_at)、idx_tasks_due(project_id,due_at,deleted_at)。
- projects.task_revision 为 BIGINT UNSIGNED NOT NULL DEFAULT 0。每次任务修改在同一事务递增；该字段为分页快照使用，不是 Project 公开 DTO 的字段。
- 成员退出/移除在同一事务清理该团队非删除任务的分配（含归档项目、已完成任务），并修改任务版本和受影响项目的 revision。其他团队不受影响。
- task.created/updated/moved/deleted/unassigned 复用 activity_logs，包含正确 team_id/project_id；软删除或离队不删审计历史。
- 迁移 9 使用 CREATE TABLE IF NOT EXISTS；迁移 10 对 ALTER 的精确后置条件作检查，支持 DDL 中断恢复，不修改旧迁移或清空数据。

## 阶段 5 已落地的约束

- comments.task_id → tasks.id，comments.author_id → users.id；不级联物理删除。任务软删除后其评论不可再通过业务 API 读取/修改，数据库行仍保留。
- content 为 VARCHAR(2000)，服务校验非空；version 为 BIGINT UNSIGNED，初始 1，CHECK(version > 0)。
- deleted_at 为 DATETIME(6) 软删除；created_at/updated_at 使用 UTC。idx_comments_task(task_id,deleted_at,id) 支持任务内倒序分页。
- 评论更新/软删除在团队 → 项目 → 任务 → 评论的固定锁序内进行，更新还带 version 条件，避免丢失并发编辑。
- 评论不改变 tasks.version、position 或 projects.task_revision。评论自己的 version 只用于评论并发控制。
- comment.created/updated/deleted 复用 activity_logs，正确填入 team_id/project_id；metadata 仅含 task_id/task_title，不存评论正文。
- 日志检索始终先限定当前团队/项目，再叠加 action、target_type、actor_id、project_id 和 UTC 时间范围。保留现有日志和索引，不回填或伪造历史。

## 阶段 6：读取聚合，不新增表

迁移版本仍为 11。团队/项目统计复用 tasks → projects 的关系，先限定 projects.team_id 与可选 tasks.project_id，排除 deleted_at 非空行，再按状态、优先级、负责人聚合。负责人昵称左连接 users，未分配以 null 表示；不依赖前端加载的成员页或任务页。

所有聚合共享当前 MySQL REPEATABLE READ 读事务；不写任务、评论、日志或项目修订号，不引入统计缓存或汇总表。现有项目、负责人及任务状态索引继续使用。团队、项目归档后仍可读取保留任务的统计。

## 阶段 7：平台审计

迁移 12 新增 platform_audit_logs：actor_id 外键关联 users，action/target_type/target_id 定位平台操作，metadata JSON 保存原因及状态变化，created_at 为 UTC；按 actor/id、created_at/id 建索引。初始化、状态变更、平台归档均写真实审计，不保存密码或会话信息。

users.status/session_version 复用既有列；状态改变和审计同事务。归档同时追加原团队日志，全部提交或全部回滚。平台统计直接读取现有表，无缓存与新汇总表，不授予普通资源访问权。
