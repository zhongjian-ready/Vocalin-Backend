# Vocalin Backend

Vocalin（窝聚）是一个面向亲密关系场景的后端服务，围绕“空间”这一核心概念提供登录鉴权、首页陪伴信息、共享记录、纪念日与愿望清单等能力。

当前版本已经重构为完整的 `Gin + GORM + Viper + Zap + Validator + JWT + Swagger` 架构，配置、日志、鉴权、路由和数据访问职责清晰，并补齐了 refresh token、统一响应体和分页规范。

## 技术栈

- Gin：HTTP 路由与中间件
- GORM：MySQL 数据访问与自动迁移
- Viper：统一配置读取
- Zap：结构化日志
- Validator：请求参数校验与自定义规则
- JWT：登录态签发与 Bearer 鉴权
- Swagger：接口文档生成与在线调试

## 目录结构

```text
cmd/
  server/      HTTP 服务入口
  seeder/      示例数据写入脚本
  cleanup/     数据库清理脚本
internal/
  app/         应用装配
  auth/        JWT 管理
  config/      Viper 配置加载
  database/    数据库连接与迁移
  handlers/    HTTP Handler
  logger/      Zap 日志初始化
  middleware/  Gin 中间件
  models/      GORM 模型
  repository/  数据访问封装
  routes/      路由注册
  service/     业务服务层
  validator/   自定义参数校验
docs/          Swagger 生成产物
```

## 快速开始

### 1. 环境准备

- Go 1.25+
- MySQL 8+
- 已创建数据库 `vocalin`

### 2. 配置环境变量

项目支持从环境变量读取配置，开发环境可以直接基于 `.env.example` 创建 `.env`。

核心配置如下：

- `SERVER_PORT`：服务端口，默认 `8080`
- `SERVER_MODE`：Gin 运行模式，建议 `debug` 或 `release`
- `DATABASE_HOST`：数据库地址
- `DATABASE_PORT`：数据库端口
- `DATABASE_USER`：数据库用户名
- `DATABASE_PASSWORD`：数据库密码
- `DATABASE_NAME`：数据库名
- `AUTH_JWT_SECRET`：JWT 密钥，生产环境必须替换为高强度随机字符串
- `AUTH_ACCESS_TOKEN_TTL`：访问令牌有效期，默认 `72h`
- `AUTH_REFRESH_TOKEN_TTL`：刷新令牌有效期，默认 `720h`
- `LOG_LEVEL`：日志级别，如 `debug`、`info`、`warn`
- `LOG_FORMAT`：日志格式，支持 `console` 与 `json`

### 3. 安装依赖

```bash
make tidy
```

### 4. 启动服务

```bash
make run
```

服务默认监听：

```text
http://localhost:8080
```

### 5. 初始化示例数据

```bash
make seed
```

Seeder 会创建两名用户、一个空间，以及照片、便签、愿望清单、纪念日等示例数据。

## 鉴权方式

服务已切换为 JWT 鉴权，不再使用 `X-User-ID` 请求头。

### 登录

```http
POST /api/auth/login
Content-Type: application/json

{
  "wechat_id": "wx_user_001",
  "nickname": "Romeo",
  "avatar_url": "https://example.com/avatar.png"
}
```

成功后返回：

```json
{
  "code": "SUCCESS",
  "message": "登录成功",
  "data": {
    "access_token": "<access-token>",
    "access_token_expires_at": "2026-05-03T12:34:56Z",
    "refresh_token": "<refresh-token>",
    "refresh_token_expires_at": "2026-05-30T12:34:56Z",
    "user": {
      "id": 1,
      "wechat_id": "wx_user_001"
    }
  }
}
```

后续请求需要在 Header 中携带：

```http
Authorization: Bearer <jwt-token>
```

### 刷新令牌

```http
POST /api/auth/refresh
Content-Type: application/json

{
  "refresh_token": "<refresh-token>"
}
```

### 登出

```http
POST /api/auth/logout
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "refresh_token": "<refresh-token>"
}
```

## 统一响应结构

除 Swagger 页面和极少数基础探针外，业务接口统一返回：

```json
{
  "code": "SUCCESS",
  "message": "操作成功",
  "data": {},
  "meta": {}
}
```

- `code`：业务状态码，例如 `SUCCESS`、`VALIDATION_ERROR`、`AUTH_UNAUTHORIZED`
- `message`：给前端和调试使用的中文提示
- `data`：业务数据主体
- `meta`：分页等附加元信息，仅在需要时返回

## 分页约定

列表接口统一支持以下查询参数：

- `page`：页码，从 `1` 开始，默认 `1`
- `page_size`：每页条数，默认 `20`，最大 `100`

响应中的 `meta` 结构如下：

```json
{
  "page": 1,
  "page_size": 20,
  "total": 53,
  "total_pages": 3
}
```

## Swagger 文档

### 在线访问

服务启动后访问：

```text
http://localhost:8080/swagger/index.html
```

### 重新生成文档

```bash
make swagger
```

这里使用 `go run` 方式调用 `swag`，不要求本地单独安装命令。

## 常用命令

```bash
make run      # 启动服务
make build    # 编译二进制
make seed     # 写入示例数据
make swagger  # 生成 Swagger 文档
make fmt      # 格式化代码
make tidy     # 整理依赖
make test     # 运行测试
```

## 设计说明

- Handler 负责请求绑定、参数校验和响应组装
- Service 负责业务规则，例如空间成员校验、首页聚合和定时便签逻辑
- Repository 负责 GORM 访问，避免 ORM 细节泄漏到上层
- Middleware 负责 JWT 解析、请求日志和跨域处理
- Config 统一由 Viper 读取，便于环境变量与配置文件双模式扩展

## 当前接口范围

- 认证：登录、刷新令牌、登出
- 空间：创建空间、加入空间、查看当前空间
- 首页：计时器、实时状态、置顶留言、首页概览
- 记录：照片、便签、愿望清单，列表接口已支持分页
- 我的：纪念日、退出空间、导出数据占位接口，纪念日列表已支持分页

## 后续建议

- 增加更多 handler 和 repository 集成测试
- 对导出数据能力接入异步任务与对象存储
