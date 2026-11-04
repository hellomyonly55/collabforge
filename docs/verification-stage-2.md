# 阶段 2 实测记录

日期：2026-10-30（Asia/Shanghai）。环境：macOS arm64、Go 1.27.3、Node 26.5.0、Docker Compose；容器 Go 1.27.3 / Node 24，本机 Chrome + Playwright。

## 已通过

- go test ./...、go vet ./...、go build ./cmd/server ./cmd/migrate。
- scripts/test-integration.sh：10 项真实 MySQL/Redis 集成测试全部通过，其中新增 4 项团队测试；独立 taskflow_test 数据库，Redis 测试实例和随机键前缀。
- 并发单次邀请码仅一个新成员加入成功；同一用户 8 个并发重试只创建一个活跃成员、扣一次次数和写一条加入日志。
- 并发转让仅一个成功，最终恰有一个 owner，teams.owner_id 匹配。
- GORM 测试回调注入 activity_logs 写入失败：团队创建、加入及计数、角色变更和所有权转让全部回滚，没有虚假成功日志。
- 所有者/管理员/普通成员权限边界、跨团队访问、平台管理员无隐式访问、成员移除/退出/重新加入、邀请码摘要保存、撤销/过期/耗尽、幂等重试、归档只读及分页默认值。
- npm run build：TypeScript 检查和生产构建通过；团队页面按路由加载。
- npm run test:e2e：最终构建上 3 项通过，包含阶段 1 认证回归和三账号真实团队流程，没有模拟接口。
- 浏览器团队流程覆盖创建、邀请码加入、未加入时拒绝访问、提升管理员、转让、退出、撤销邀请码、日志展示及归档；390px 团队列表和成员详情无页面横向溢出，成员表在表格区域滚动。
- 邀请码撤销后当前页面的原始码清除；邀请截图遮盖原始码，文档截图不含邀请或会话凭证。
- docker compose config --quiet、最终 Compose 镜像构建与启动通过，迁移版本 6，健康检查 MySQL/Redis/migrations 全为 ok。
- OpenAPI YAML 解析及内部引用检查通过。

## 交付状态

正常服务保留运行在 http://localhost:8080。测试服务单独停止，没有删除演示数据卷或批量删除文件。测试使用真实随机邮箱账号；浏览器团队流程创建的团队在流程末尾归档，记录保留。

截图：[成员管理](images/stage-2-members.png)、[移动端成员](images/stage-2-members-mobile.png)、[移动端团队列表](images/stage-2-list-mobile.png)。

阶段 2 完成。项目、任务、评论、统计及管理员后台仍待后续模块；基础团队动态现可查询真实日志，完整日志检索在阶段 5 完善。下一模块为阶段 3：项目管理。
