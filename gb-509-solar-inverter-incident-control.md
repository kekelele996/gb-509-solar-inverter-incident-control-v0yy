请生成 `solar-inverter-incident-control`「光伏逆变器故障处置控制」Go 全栈项目，面向新能源场站管理逆变器、故障事件、保护策略和远程处置确认。项目重点是设备状态与安全操作，不做电商、库存或泛化看板。

## 项目主要需求

复杂度下限：核心实体不少于 3 个、核心页面不少于 4 个、横切关注点不少于 2 个、共享前端组件不少于 3 个、自定义 hooks/utils 不少于 2 个、后端中间件不少于 2 个。

### 核心实体

`SolarSite`（场站与并网信息）、`InverterUnit`（逆变器状态）、`FaultEvent`（故障等级与证据）、`MitigationAction`（处置策略与执行确认）贯穿数据库、Go 分层和前端。

### 核心页面

`/sites` 场站；`/inverters` 设备；`/faults` 故障处置；`/actions` 策略确认；`/audit` 审计。`SeverityTag` 在设备和故障页共用，`ActionDrawer` 在故障和策略页共用。

### 横切关注点

RBAC 联动角色表、后端中间件、前端路由守卫和操作显隐；远程动作必须双重确认并写审计；加入全局错误处理、请求追踪、Redis 限流。

### 共享枚举/组件

同步 `InverterState`（online/warning/tripped/isolated）与 `FaultState`（open/acknowledged/mitigated/closed）。共享 `StatusBadge`、`SeverityTag`、`ConfirmDialog`，hooks 为 `useAuth`、`usePolling`。

### 技术与规模要求

前端 React 18 + TypeScript + Vite + Ant Design；后端 Go 1.22 + Gin + GORM；PostgreSQL、Redis。目标 2800–4000 行、28–40 个 `.go` 文件。

### 文件结构强制清单

前端必须有 `api/stores/types/components/common/hooks/pages/router/utils`；后端必须有 `model/dto/repository/service/handler/router/middleware/constants/util`，禁止合并职责。

### 结构红线

严禁合并职责到单一文件；故障处置必须拆分 model、service、handler、middleware 和前端模块。

### 部署与交付

根目录必须提供 `docker-compose.yml`（顶层 `name: solar-inverter-incident-control`，且不写 `version:`）、`.env` 和 `.env.example`（均含 `COMPOSE_PROJECT_NAME=solar-inverter-incident-control`）、`README.md`、`frontend/Dockerfile`、`backend/Dockerfile` 和 `frontend/nginx.conf`。前端端口 `18509`、后端端口 `19509`；Nginx `/api` 代理、数据库 healthcheck、命名卷和 `condition: service_healthy` 齐全，提供真实 `/healthz`、Git 初始化，中文目录可启动。
