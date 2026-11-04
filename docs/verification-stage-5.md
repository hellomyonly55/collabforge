# 阶段 5 实测记录

> 此为历史阶段记录。阶段 6 已交付数据统计，当前运行和验收见 [阶段 6 验收](acceptance-stage-6.md)。

日期：2026-10-31（Asia/Shanghai）。运行、可执行 API、浏览器操作和验收标准见 [阶段 5 验收](acceptance-stage-5.md)。

## 已执行检查

| 检查 | 结果 |
|---|---|
| Go 1.27.3，go test ./... | 通过；未开启集成开关时，外部依赖测试明确跳过 |
| go vet ./... | 通过 |
| scripts/test-integration.sh | 独立真实 MySQL/Redis 的 28 项全部通过，11.077 秒 |
| npm run build | Vue TypeScript 与 Vite 生产构建通过 |
| Docker 内服务与迁移命令构建 | 通过 |
| scripts/test-e2e.sh，系统 Chrome | 全部 8 项浏览器回归通过，21.3 秒 |
| 最终评论与日志定向回归 | 1 项通过，4.7 秒；截图显示实际评论正文与日志记录 |
| OpenAPI 0.6.0 | YAML 解析、全部内部 $ref 校验通过 |
| docker compose up --build -d | 原演示环境升级成功，migrate 为 Exited (0) |
| schema_migrations | 10 → 11 |
| GET /api/v1/health/ready | HTTP 200，mysql/redis/migrations 均为 ok |

Go 测试使用 taskflow_test 与独立 Redis；浏览器测试通过 compose.e2e.yaml 的完整真实应用栈、独立 MySQL/Redis，端口 18080。测试停止后，8080 主应用保持运行。没有浏览器假接口或内存假业务存储。

## 新增 5 项真实集成测试

- TestCommentLifecyclePermissionsAndValidation：实际发表评论与持久化、作者编辑/删除、owner 删除他人但不可编辑、普通成员越权、平台 admin 无团队特权、跨团队 404、内容/未知字段/版本校验、旧版本 409、软删除及任务版本不变。
- TestCommentArchiveDeletionAndMembership：项目/团队归档只读、任务软删除后评论不可读写、移除成员后访问失效，数据库历史保留。
- TestCommentAuditRollbackAndConcurrency：注入日志失败后创建/编辑/删除均回滚；两次相同版本编辑只能一个成功，失败请求不留成功事件；日志不含评论正文。
- TestCommentPaginationAndActivityFilters：23 条真实评论分页 20+3；日志 action/target_type/actor_id/project_id/from/to 组合筛选及非法参数、项目/团队隔离、历史离队作者仍可查询。
- TestCommentRemovalRaceAndRateLimit：评论发表与移除并发保持权限边界；实际 Redis 每用户 120 次发表请求限流返回 429。

此前 23 项认证、团队、项目、任务和迁移测试全部一并通过。单元测试新增上海日期边界转换 UTC、非法 RFC3339 时间及倒置区间校验。

## 浏览器实测流程

新增端到端测试使用三个真实账号，并复制作者 Cookie 到第二窗口验证并发编辑；不预置管理员密码，不往主库植入演示记录。

1. 注册/登录 → 建团队 → 邀请加入 → 建项目/任务 → 任务详情发表真实评论，刷新后抽屉可读。
2. HTML 标签按原文显示，img 元素数量为零，onerror 没有执行。
3. 作者编辑自己的评论；团队 owner 看不到编辑入口，但可删除。作者第二窗口旧版本保存返回冲突，编辑状态取消并显示服务器最新内容。
4. 删除后列表空；通过真实 API 再创建 23 条评论，页面第一页 20、第二页 3。
5. 项目日志按评论编辑、目标、作者与当天日期过滤得到准确 2 条，不包含评论正文；重置/翻页正确。团队日志按项目和任务创建过滤得到 1 条，可打开真实任务。
6. 外部账号直接访问任务及日志页面得到无权限/404，不显示评论区。
7. 归档后评论可读且写入口消失；390px 评论和日志页面无整页横向溢出。
8. 原认证、团队、项目、任务拖拽/冲突/分页等 7 项完整回归同时通过。

## 原环境数据保留

升级前：users=13、teams=3、projects=0、tasks=0、activity_logs=20，版本 10。

升级后：上述计数均保持；新 comments 表为 0 行，版本 11。没有向主库写入评论测试数据，也未删除业务历史。登录自己的账号创建项目、任务即可使用新模块。

## 真实页面截图

来自独立浏览器测试环境的真实 API 数据；主环境没有预置这些账号/评论。

- [任务抽屉评论](images/stage-5-comments-drawer.png)
- [项目操作日志查询](images/stage-5-activity-project.png)
- [390px 评论](images/stage-5-comments-mobile.png)
- [390px 操作日志](images/stage-5-activity-mobile.png)

## 限制

纯文本、倒序分页，无实时推送、楼层回复、Markdown、@ 提及或附件。历史已删除资源的详情链接会返回 404。评论软删除不等于数据库物理抹除；日志不复制评论正文。数据统计和管理员后台尚未实现，下一阶段为真实任务统计。
