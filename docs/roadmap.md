# 分模块实施计划

阶段 0：工程、Compose、健康检查、版本化迁移。已完成。
阶段 1：用户认证、个人设置、会话与限流。已完成。
阶段 2：团队管理、成员权限、邀请码、所有权及基础团队动态。已完成。
阶段 3：项目列表、创建/编辑/归档、继承团队权限及项目动态。已完成。
阶段 4：任务 CRUD、单负责人、四列看板、拖拽、筛选、我的任务、版本冲突和离队清理。已完成。
阶段 5：评论增删改、作者/管理员权限、评论版本冲突、团队/项目日志检索与分页。已完成。
阶段 6：团队/项目真实任务聚合、状态/优先级/负责人图表、逾期和完成率。已完成。
阶段 7：管理员 CLI、用户禁用/启用与会话撤销、团队查看/归档、平台统计与审计。已完成。
阶段 8：连续三账号协作、真实依赖故障恢复、总验收及交付文档。已完成，首版范围交付。

以下模块均已按顺序交付，每模块包含前后端联调、测试、运行方法及验收记录：

1. 认证：注册、登录/退出、资料、修改密码、bcrypt、Redis 会话、限流、路由守卫。
2. 团队：创建/编辑/归档、成员权限、邀请码、退出和转让；本阶段起同步写操作日志。
3. 项目：创建/编辑/归档、团队隔离、只读归档。
4. 任务：单负责人、四列看板、拖拽、筛选、我的任务、乐观锁与项目锁。
5. 评论和日志：评论权限、日志列表及分页。
6. 统计：按状态、优先级、负责人和逾期汇总真实任务。
7. 后台：安全初始化管理员、用户禁用并撤销会话、团队查看/归档、平台统计。
8. 最终回归：双普通账号和管理员端到端流程、Playwright、项目演示及校招说明。

## 已约定的业务 API（认证、团队、项目、任务与看板、评论、操作日志、统计和后台已实现，其余待开发）

统一前缀 `/api/v1`。ID 为字符串；列表返回 items/page/page_size/total，默认 20、最大 100。创建 201，删除 204；失败使用稳定 error.code/message/request_id。

| 模块 | 路由 |
|---|---|
| 认证 | POST /auth/register、/auth/login、/auth/logout；GET/PATCH /me；PUT /me/password |
| 团队 | GET/POST /teams；GET/PATCH /teams/:teamId；POST /teams/:teamId/archive、/leave、/transfer-ownership |
| 成员 | GET /teams/:teamId/members；PATCH/DELETE /teams/:teamId/members/:userId |
| 邀请 | GET/POST /teams/:teamId/invites；DELETE /teams/:teamId/invites/:inviteId；POST /teams/join |
| 项目 | GET/POST /teams/:teamId/projects；GET/PATCH /projects/:projectId；POST /projects/:projectId/archive |
| 任务 | GET/POST /projects/:projectId/tasks；GET/PATCH/DELETE /tasks/:taskId；GET /me/tasks |
| 看板 | GET /projects/:projectId/board；POST /tasks/:taskId/move |
| 评论 | GET/POST /tasks/:taskId/comments；PATCH/DELETE /comments/:commentId |
| 日志 | GET /teams/:teamId/activities、/projects/:projectId/activities |
| 统计 | GET /teams/:teamId/statistics、/projects/:projectId/statistics |
| 后台 | GET /admin/statistics、/admin/users、/admin/teams；PATCH /admin/users/:userId/status；POST /admin/teams/:teamId/archive |

不提供假后端、不预置默认管理员密码；首版不含实时推送、附件、邮件、自定义流程。
