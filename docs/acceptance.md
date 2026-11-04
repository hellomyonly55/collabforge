# 阶段 0 验收

> 此为历史阶段记录。当前阶段 4 已交付任务与看板，最新运行和验收见 [阶段 4 验收](acceptance-stage-4.md)。

阶段 1 验收见 [认证验收](acceptance-stage-1.md)。当前系统状态页为 http://localhost:8080/health，根地址进入登录或工作区。

## 正常流程

1. 配置 .env，运行 docker compose up --build -d。
2. docker compose ps -a：MySQL/Redis/backend 启动，migrate 退出码为 0。
3. 打开 http://localhost:8080，页面请求真实 /api/v1/health/ready，三个依赖均为正常。
4. 点击刷新，时间和 request_id 更新。
5. curl 请求 live 为 200/alive；ready 为 200/ready；响应头 X-Request-ID 与 JSON 一致。
6. docker compose run --rm migrate 重复迁移成功；版本 1 只存在一条。
7. docker compose down 后重新启动，迁移记录保留，数据卷没有被删除。

## 故障流程

```sh
docker compose stop redis
curl -i http://localhost:8080/api/v1/health/ready
curl -i http://localhost:8080/api/v1/health/live
```

ready 应为 503，redis 为 unavailable；live 仍为 200。页面刷新显示真实故障。恢复：

```sh
docker compose start redis
```

等待恢复后刷新应正常。同样可暂停 MySQL，验证 mysql/migrations unavailable，再启动恢复。后端停止时，页面应展示错误提示，不能残留绿色成功状态。

## 来源与未实现接口

```sh
curl -i -X POST -H 'Origin: https://attacker.example' http://localhost:8080/api/v1/not-implemented
curl -i http://localhost:8080/api/v1/teams
```

分别为 403 和 404；JSON 不含 SQL、密码或堆栈。

## 自动检查

- go test ./...：配置、健康检查成功/故障、请求 ID、来源校验。
- TASKFLOW_INTEGRATION=1 go test -v ./tests/integration：真实依赖与迁移幂等。
- go vet ./...；go build ./cmd/server ./cmd/migrate。
- npm run build：TypeScript 检查和生产构建。
- docker compose config --quiet：Compose 结构和变量校验。

## 后续验收

认证模块将补充密码、会话与路由守卫测试；任务模块将补充权限/并发测试；完整业务落地后增加 Playwright 双账号端到端测试。阶段 0 不声称已通过这些尚未实现的测试。
