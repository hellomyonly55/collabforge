# 阶段 1 实测记录

日期：2026-10-30（Asia/Shanghai）。环境：macOS arm64，临时 Go 1.27.3、Node 26.5.0，Docker Compose；容器 Go 1.27.3 / Node 24。

## 通过的检查

- go test ./...：配置、HTTPS Cookie 配置、代理配置、密码字节边界、bcrypt 比较、邮箱与昵称校验、健康检查等。
- go vet ./...；go build ./cmd/server ./cmd/migrate。
- scripts/test-integration.sh：6 项集成测试通过，独立 MySQL/Redis，taskflow_test 数据库，随机 Redis 命名空间。
- 集成覆盖：注册唯一约束、DTO 不泄露凭据、密码摘要、双会话撤销、登录凭证轮换、退出失效、密码更改、账号禁用、未知凭证、1 秒测试会话自然到期、数据库版本强制失效、并发密码更新单一胜者、注册/登录限流、来源校验、Secure Cookie、迁移重复运行。
- npm run build：TypeScript 校验及生产构建通过。
- npm run test:e2e：本机 Chrome / Playwright 2 项通过，真实注册→登录→资料保存→刷新→两设备登录→改密→所有旧会话失效→新密码登录→退出，以及 390px 移动端无横向溢出。
- Compose 配置校验与最终镜像构建通过，迁移版本 2，既有阶段 0 数据卷原地升级。
- OpenAPI YAML 解析及内部引用检查通过。

## 真实故障注入验收

- 停止 Redis：/me 与登录 API 返回 503，已登录页面导航进入可重试的连接错误页。
- 恢复 Redis：重新连接后同一会话正常恢复，没有将依赖故障误判为登录凭据错误。
- 停止 MySQL：/me 与登录 API 返回 503；个人资料保存显示失败。
- 恢复 MySQL：原用户资料保持不变，原会话正常，健康检查全部恢复。

故障验收使用真实容器停止与启动，没有模拟成功响应。正常服务保留运行在 http://localhost:8080；独立测试服务在验收后停止，未删除演示数据卷。

截图：[登录页](images/stage-1-login.png)、[个人设置](images/stage-1-settings.png)。

## 交付边界

阶段 1 完成。团队、项目、任务、评论、日志、统计和管理员后台仍待后续模块实现。浏览器自动验收创建了随机邮箱的真实测试账号并保留记录，没有默认登录账号。没有执行批量文件删除。
