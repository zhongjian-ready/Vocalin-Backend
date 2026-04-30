# Copilot 说明

## 项目背景

- 这是一个 Go 后端项目，模块入口位于 `cmd/`，HTTP 服务入口是 `cmd/server/main.go`，辅助命令包括 `cmd/seeder/main.go` 和 `cmd/cleanup/main.go`。
- 业务主代码位于 `internal/`，当前主要目录包括 `app/`、`auth/`、`config/`、`database/`、`handlers/`、`middleware/`、`models/`、`repository/`、`response/`、`routes/`、`service/` 和 `validator/`。
- 当前应用采用 `Gin + GORM + Viper + Zap + Validator + JWT + Swagger` 架构，应用装配集中在 `internal/app`，路由注册集中在 `internal/routes`。
- 接口鉴权基于 JWT，受保护接口通过 `internal/middleware/auth_middleware.go` 解析 Bearer Token，不再使用 `X-User-ID` 作为正式鉴权方案。
- 项目支持通过环境变量和 `.env` 加载配置；涉及端口、数据库、JWT 或日志配置时，优先沿用 `internal/config` 的现有加载方式。

## 全局要求

- 所有自动生成和修改的代码都必须遵守仓库编码规范，以及 `.github/instructions/` 下与当前文件类型匹配的规则。
- 保持修改尽可能小，并聚焦于需求本身，不重构无关代码。
- 优先复用现有实现，不重复创建已有模式的 handler、service、repository、middleware 或路由逻辑。
- 优先与附近已有代码保持一致，不引入无必要的新抽象、新目录层级或跨层调用。
- 不保留未使用的参数、变量、表达式、方法、导入和资源声明。
- 命名必须语义化，禁止使用无意义命名。
- 除非逻辑确实不直观，否则避免添加注释。
- 不要为文件添加作者、时间等头部注释信息。
- 不要覆盖用户已有修改或无关改动。

## 文件组织与命名

- HTTP 入口、脚本和命令放在 `cmd/` 对应目录下，不要把启动逻辑散落到 `internal/`。
- Handler 放在 `internal/handlers/`，业务编排放在 `internal/service/`，数据访问放在 `internal/repository/`，模型放在 `internal/models/`，中间件和路由分别放在 `internal/middleware/` 与 `internal/routes/`。
- 配置、数据库、日志、鉴权和校验等基础能力分别放在 `internal/config/`、`internal/database/`、`internal/logger/`、`internal/auth/` 和 `internal/validator/`，不要把这些职责混入 handler 或 service。
- Swagger 产物位于 `docs/`，仅在接口契约或 Swagger 注释变更时更新。
- 测试文件与被测包就近放置，优先沿用 `internal/**/_test.go` 的现有结构和命名方式。

## Go 后端开发约束

- 优先沿用当前的分层结构：handler 负责请求绑定、参数校验和响应组装，service 负责业务规则，repository 负责 GORM 访问，不要把数据库细节泄漏到 handler。
- 涉及鉴权、令牌、会话或用户上下文时，先检查 `internal/auth`、`internal/middleware` 和相关 service 的现有实现，避免绕过 JWT 流程。
- 涉及接口返回时，优先沿用 `internal/response` 的统一响应结构，不要在 handler 中手写风格不一致的响应体。
- 涉及分页时，优先复用现有分页参数与 `meta` 结构，不要引入新的字段命名或分页语义。
- 涉及环境变量、数据库或日志初始化时，先检查 `internal/config`、`internal/database`、`internal/logger` 和 `internal/app` 的现有装配流程，避免在局部代码中重复初始化基础依赖。
- 涉及新增或调整接口时，优先在对应 handler 上维护 Swagger 注释，并在需要时重新生成 `docs/`。

## 测试与验证

- 修改完成后，至少对变更文件做一次快速错误检查，并对受影响包执行最小范围验证。
- 如果改动影响业务逻辑、鉴权、中间件、服务层或 HTTP 行为，优先补充或更新对应包下的测试，例如 `internal/auth/`、`internal/handlers/` 或 `internal/service/`。
- 涉及 Go 代码变更时，优先运行 `gofmt -w` 格式化受影响文件；涉及接口契约变更时，再决定是否执行 `make swagger`。
- 当前仓库已有 `internal/auth/jwt_test.go`、`internal/handlers/common_test.go`、`internal/service/auth_service_test.go` 和 `internal/service/record_service_test.go`，新增测试时优先参考这些结构。

## 工作方式

- 先搜索现有实现，再决定是否新增 handler、service、repository 或中间件。
- 先从最接近需求的路由入口、handler、service 或仓储层入手，避免做大范围无关探索。
- 变更后优先做最小范围验证，再决定是否继续扩展修改。
