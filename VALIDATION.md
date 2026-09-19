# 验收记录

- 日期：2026-08-22
- 静态检查：Go 1.22 下 `go test ./...`、`go build ./...` 通过；React 类型检查和 Vite 生产构建通过；`docker compose config --quiet` 通过。
- 容器启动：PostgreSQL、Redis、backend、frontend 从空数据卷启动成功并达到 healthy，`GET /healthz` 返回 200。
- API 流程：管理员登录、概览、4 个实体列表、创建场站与处置动作、合法状态迁移、会话、脱敏运行配置、审计列表及审计汇总均通过。
- RBAC 与远程确认：viewer 写操作返回 403；处置动作缺少 `confirmed` 标记时返回 422；经过普通迁移确认和勾选式远程二次确认后迁移成功并写入审计。
- 内置 Browser：验证光伏场站、逆变器、故障事件、处置动作、审计记录 5 个页面；`SeverityTag` 在逆变器与故障页正常展示；`ActionDrawer` 的禁用态、勾选确认、执行、状态刷新和审计回显正常；控制台 0 error / 0 warning，桌面截图未见遮挡或错位。
- 规模：3019 行 Go 功能代码，38 个非测试 `.go` 文件。
- 清理：验收完成后执行 `docker compose down -v --remove-orphans`，清除本项目容器和数据卷。
