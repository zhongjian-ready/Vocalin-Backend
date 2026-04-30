---
description: 'Use when editing Go backend files. Enforce repository layering, API conventions, and validation rules.'
applyTo: '**/*.go'
---

# Go 后端规范

- 优先沿用当前项目的 `Gin + GORM + Viper + Zap + Validator + JWT + Swagger` 架构，不无故引入新的 Web 框架、ORM、配置系统或鉴权方案。
- HTTP handler 放在 `internal/handlers/`，负责请求绑定、参数校验、调用 service 和返回统一响应；不要在 handler 中直接操作数据库。
- 业务规则放在 `internal/service/`，数据访问放在 `internal/repository/`；新增逻辑时优先复用 `repository.Store` 和现有 service 装配方式。
- 路由注册集中在 `internal/routes/`，中间件放在 `internal/middleware/`；新增接口优先沿用现有分组、鉴权和中间件链路。
- 涉及鉴权或用户上下文时，优先复用 `internal/auth`、`AuthMiddleware` 和现有 token 管理逻辑，不要回退到 `X-User-ID` 或其他临时方案。
- 涉及统一响应时，优先复用 `internal/response/`；涉及分页时，沿用当前 `page`、`page_size` 和 `meta` 结构。
- 涉及配置、数据库、日志或应用初始化时，优先检查 `internal/config/`、`internal/database/`、`internal/logger/` 和 `internal/app/`，避免在局部代码中重复初始化基础依赖。
- 涉及模型字段、数据表结构或关联关系时，优先修改 `internal/models/` 和相关 repository/service，并考虑自动迁移与 seed 数据影响。
- 涉及接口契约调整时，优先维护 handler 上的 Swagger 注释；只有在契约变化时才更新 `docs/` 生成产物。
- 修改 Go 代码后保持导入整洁，不保留未使用的变量、参数、方法或包引用。
- 新增测试优先放在对应包下，沿用现有 `_test.go` 结构；改动 service、auth、middleware 或 handler 时，优先补充相邻包测试。
- 完成改动后至少执行一次最小范围验证；如修改 Go 文件，优先运行 `gofmt -w` 和对应包的 `go test`，必要时再运行 `go test ./...` 或 `make swagger`。
