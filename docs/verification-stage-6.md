# 阶段 6 实测记录

> 此为历史阶段记录。阶段 7 已交付管理员后台，当前运行和验收见 [阶段 7 验收](acceptance-stage-7.md)。

日期：2026-10-31（Asia/Shanghai）。运行、API、测试命令及手工验收标准见 [阶段 6 验收](acceptance-stage-6.md)。

## 执行结果

| 检查 | 结果 |
|---|---|
| Go 1.27.3，go test ./...、go vet ./... | 通过；普通运行明确跳过外部依赖测试 |
| scripts/test-integration.sh | 32 项真实 MySQL/Redis 集成测试通过，16.190 秒 |
| npm run build | Vue 类型检查及 Vite 生产构建通过 |
| scripts/test-e2e.sh，系统 Chrome | 9 项完整浏览器回归通过，23.1 秒 |
| Docker 服务与迁移构建 | 通过 |
| docker compose up --build -d | 原演示环境更新成功，migrate 为 Exited (0) |
| GET /api/v1/health/ready | HTTP 200，mysql/redis/migrations 均为 ok |
| OpenAPI 0.7.0 | YAML 解析与内部引用校验通过 |

## 新增 4 项真实集成测试

- TestStatisticsAggregationAndTaskChanges：空项目零值、四状态/四优先级、负责人/未分配、今日与过去日期、已完成不逾期、移动/编辑/删除后刷新、完成率取整、团队包含归档项目。
- TestStatisticsAuthorizationArchiveAndMembership：普通成员可读、外部平台 admin 无隐式权限、跨团队隔离、项目/团队归档仍可读、离队权限失效及分配清理、非法和不存在 ID。
- TestStatisticsIncludesAllPages：创建 105 条真实任务，团队/项目总数均为 105，不受任务列表或看板分页限制。
- TestStatisticsSnapshotAndDatabaseFailure：首个聚合后实际提交一条任务，当前响应仍保持原读快照，下一请求可见新任务；注入聚合数据库错误后返回 503，不泄露错误细节或返回虚假零值。

此前 28 项认证、团队、项目、任务、评论、日志和迁移测试同时回归通过；单元测试补充 0/0、1/3、2/3、3/3 的完成率边界。

## 浏览器实测

统计测试通过真实 API 创建账号、团队、项目及四状态任务；空项目先显示 0，然后刷新得到总数 4、完成 1、未完成 3、逾期 2、完成率 25%。在真实任务页面把逾期任务移动到已完成后，统计变为完成 2、逾期 1、完成率 50%；软删除后总数 3、完成率 33.3%。

另建一个归档项目及保留任务，团队汇总为 4 项；390px 卡片、图表和负责人表无整页横向溢出。未登录访问转登录并且 API 返回 401；归档团队后统计仍可查看。页面无 JavaScript 错误。其他 8 项既有浏览器流程一并通过。

浏览器使用独立 18080 应用与独立 MySQL/Redis，无 mock；Go 使用独立 taskflow_test。截图里的任务均经真实 API 保存，不是演示主库中的预置数据。

## 数据与部署

本阶段不新增迁移或环境变量，数据库版本仍为 11。升级前主库计数：users=13、teams=3、projects=0、tasks=0、comments=0、activity_logs=20；统计模块只读，不创建演示任务。升级后上述计数全部保持不变，迁移服务报告版本 11；统计未登录请求为 401，测试依赖已停止，主应用保持 8080 运行。

## 真实截图

- [项目统计](images/stage-6-statistics-project.png)
- [390px 团队统计](images/stage-6-statistics-mobile.png)

## 当前限制

无历史趋势、导出、日期过滤或实时推送；负责人分组全部返回。图表百分比独立保留一位小数，数量不做近似。管理员后台尚未实现，下一阶段 7 完成。
