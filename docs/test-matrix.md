# 功能与测试覆盖

| 能力 | 真实集成 | 浏览器 | 最终流程 |
|---|---|---|---|
| 注册/登录/资料/改密/退出 | auth_test.go | auth.spec.ts | 两普通账号实际注册 |
| 团队成员/邀请/owner 规则 | team_test.go | teams.spec.ts | 邀请、加入 |
| 项目隔离/归档 | project_test.go | projects.spec.ts | 创建项目 |
| 任务分配/移动/版本/排序 | task_test.go | tasks.spec.ts | B 的任务、拖拽、验收 |
| 评论权限/版本/审计回滚 | comment_test.go | comments.spec.ts | B 评论、A 查看 |
| 日志范围/筛选 | project/comment 测试 | projects/comments | 查看实际事件 |
| 统计/零值/105 条全量/快照 | statistics_test.go | statistics.spec.ts | 完成率 100% |
| 管理员/禁用/归档/Redis 清理失败 | admin_test.go | admin.spec.ts | C 禁用 B、B 被拒绝 |
| 迁移/依赖/重试 | health/project/task 测试 | 健康页 | 暂停 MySQL、停止 Redis、恢复 |
| 初始化不覆盖配置 | Node init-env.test.mjs | 不适用 | 新配置 0600 |
| API/文档引用 | cmd/checkdocs | 不适用 | 总验收执行 |

Go 普通测试会明确跳过外部依赖，不能把它当成真实集成通过。scripts/test-integration.sh 开启开关并连接独立测试库。scripts/test-e2e.sh 是 10 个模块浏览器用例；scripts/test-workflow.sh 另起全新测试状态完成一个完整三账号用例，包含真实依赖故障恢复。

所有浏览器测试通过真实 HTTP/数据库，不用前端 mock。注入日志/查询失败的 Go 测试用于证明事务与错误路径；最终流程额外停止/暂停实际测试容器，证明真实依赖故障行为。主环境不参与破坏性故障测试。
