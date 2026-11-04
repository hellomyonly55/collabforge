# TaskFlow 1.0 最终实测记录

日期：2026-10-31（Asia/Shanghai）。命令从项目根目录执行，使用 Go 1.27.3、Node 26.5.0、系统 Chrome 和运行中的 Docker。运行与手工标准见 [最终验收](acceptance-stage-8.md)。

## 实际执行结果

总验收命令：

```sh
PATH="/tmp/taskflow-tools/go/bin:$PATH" \
PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' \
./scripts/verify.sh
```

本机 Go 使用临时工具目录；其他环境安装 Go 并加入 PATH 后无需该前缀。脚本最终返回 0 并输出 `PASS: TaskFlow release verification completed.`。

| 检查 | 实际结果 |
|---|---|
| 私有配置初始化 | 1 项 Node 测试通过；随机密码不同、0600、已有配置不覆盖 |
| Compose | config --quiet 通过 |
| Go | 全包 go test、go vet、四个 cmd 构建通过；外部依赖测试在此步明确跳过 |
| API 与文档 | 48 个 operationId、377 个内部引用、25 个 Markdown 文件校验通过 |
| 发布契约 | 独立单元测试证明 48 个公开操作与 Gin 注册路由一致 |
| 前端 | Vue 类型检查与 TaskFlow 1.0.0 生产构建通过（Vite 7.3.7） |
| 真实集成 | 37 项 MySQL/Redis 集成测试全部通过 |
| 模块浏览器回归 | 10 个用例全部通过 |
| 连续三账号流程与实际故障恢复 | 1 个完整用例通过，9.5 秒（场景执行 9.0 秒） |

不是以普通 go test 的跳过结果代替集成通过。总验收另外启动真实测试数据库与 Redis；完整流程另起全新应用状态，没有降低生产限流，没有浏览器 mock。

## 连续闭环证据

两个普通账号实际在页面注册登录；A 建团队、生成邀请，B 加入；A 在页面建项目/任务并分配给 B；B 的“我的任务”出现该任务，拖到进行中后刷新仍保留；B 发表评论，A 能读取并验收为已完成；日志出现评论/移动事件，统计完成率 100%；C 登录真实 CLI 初始化的管理员，后台填写原因禁用 B；B 刷新转登录且 API 返回 401。

该用例在同一条连续流程完成，未把分散成功的模块测试当成完整演示。

## 实际依赖故障

仅作用于 taskflow-e2e 测试栈：暂停 MySQL，ready 为 503 且 mysql=unavailable，live 保持 200；unpause 后 ready 恢复 200。停止 Redis，ready 为 503、live 为 200，工作区进入依赖错误页；重启 Redis 后 ready 恢复 200。测试 Redis 无持久化，会话实际丢失，重试后回登录，未伪造登录成功。

测试 finally 恢复暂停/停止状态，脚本随后停止独立测试容器；总验收也停止 Go 测试 MySQL/Redis。主应用 8080 未被故障注入。

## 核查与交付

审阅了真实错误响应、Origin 校验、成员与平台角色边界、密码/会话处理、任务与评论版本、归档只读、日志事务和统计口径。没有添加线上规模、TPS 或高可用声明。新建文档检查命令使契约与本地链接可重复验证。

主库更新前计数为 users=14、teams=3、projects=0、tasks=0、comments=0、activity_logs=20、platform_audit_logs=1，迁移版本 12；最终部署后所有上述计数保持不变。本阶段没有创建主库演示账号或修改管理员凭据。

主环境已执行 docker compose up --build -d，migrate=Exited(0)，版本仍为 12；ready 为 200，MySQL/Redis/migrations 均为 ok。MySQL 容器因优雅停止配置更新而重建，持久卷中的上述数据未丢失；测试 MySQL/Redis 均 Exited(0)。既有私有管理员凭据的 0600 权限、实际登录和后台读取已复核，验证后退出会话，密码未输出。

## 真实截图

来自独立完整流程的真实页面，不是主库预置数据：

- [任务验收后完成率 100%](images/stage-8-workflow-statistics.png)
- [实际 Redis 故障提示](images/stage-8-dependency-error.png)

模块截图保留在各阶段记录中；邀请码截图已遮挡，凭据文件和 .env 不出现在截图或交付材料中。

## 公开范围与限制

首版范围已完成；无实时推送、附件/邮件、趋势历史、在线提权、恢复归档/删除或平台审计检索页面。排序及负责人聚合适合中小项目，尚未做大规模压测。完整使用步骤见 README，三账号手动流程见 demo.md，学习解释见 learning-guide.md。
