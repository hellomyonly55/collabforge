# 阶段 0 实测记录

日期：2026-10-30（Asia/Shanghai）。

环境：macOS arm64；临时校验工具 Go 1.27.3；宿主 Node 26.5.0；Docker 29.5.3 / Compose 5.1.4；容器前端 Node 24。Go 官方归档下载后校验 SHA256，工具位于 /tmp/taskflow-tools/go，不修改系统 Go 安装。

## 已通过

- go test ./...：配置、健康检查、请求 ID 和来源检查；外部依赖测试默认跳过。
- go test -race ./internal/...：通过。
- go vet ./... 和 go build ./cmd/server ./cmd/migrate：通过。
- TASKFLOW_INTEGRATION=1 go test -v ./tests/integration：实际连接 Compose MySQL/Redis，重复迁移两次，版本记录唯一，真实 ready 返回 200。
- npm run build：类型检查和生产构建通过；按需注册 Element Plus 组件后主 JS 约 161 kB（gzip 约 62 kB）。
- docker compose config --quiet：通过。
- docker compose up --build -d：镜像构建、MySQL/Redis、迁移和 Nginx 启动通过；迁移退出码 0，后端健康。
- Chrome + Playwright 实际浏览器检查：三个依赖正常；刷新后 request_id 更新；无页面 JS 错误；390px 窄屏无横向溢出。
- 停止真实 Redis：ready 返回 503，redis 为 unavailable；live 保持 200；浏览器显示部分依赖不可用。恢复后 ready 返回 200。
- 跨来源 POST 返回 403；尚未实现的 /teams 返回 404。

截图：images/stage-0-desktop.png、images/stage-0-mobile.png。

浏览器检查使用临时安装的 Playwright 和本机 Chrome 执行；后续业务端到端测试将作为独立项目测试套件加入，当前不声称存在完整业务 E2E。

## 当前交付边界

只创建 schema_migrations 表，无业务种子数据。登录、团队、项目、任务、评论、日志、统计和后台尚未实现。已生成仅本地使用、权限 0600、被 gitignore 排除的 .env，数据库随机密码不写入文档。服务保留运行，可在本地 8080 访问。
