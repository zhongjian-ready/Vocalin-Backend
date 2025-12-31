# Vocalin Backend (窝聚)

## 简介

Vocalin (窝聚) 是一个专注于亲密关系（情侣、密友、家人）的封闭式社交应用后端。

## 功能模块

1.  **核心逻辑**：基于“房间/空间”的分组模式。
2.  **首页**：陪伴计时器、实时状态、置顶留言、最近动态。
3.  **记录**：共享相册、时光留言板、愿望清单。
4.  **我的**：个人设置、空间管理、纪念日提醒。

## 技术栈

- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: MySQL
- **ORM**: GORM
- **Docs**: Swagger

## 快速开始

### 1. 环境准备

- 安装 Go
- 安装 MySQL，并创建数据库 `vocalin`

### 2. 配置

项目默认使用环境变量配置，也可以修改 `internal/config/config.go` 中的默认值。
主要环境变量：

- `DB_USER`: 数据库用户名 (默认 root)
- `DB_PASSWORD`: 数据库密码 (默认 password)
- `DB_HOST`: 数据库地址 (默认 127.0.0.1)
- `DB_PORT`: 数据库端口 (默认 3306)
- `DB_NAME`: 数据库名称 (默认 vocalin)

### 3. 运行

```bash
# 安装依赖
go mod tidy

# 运行服务
make run
# 或者
go run cmd/server/main.go
```

### 4. API 文档

服务启动后，访问：
http://localhost:8080/swagger/index.html

### 5. 生成 Swagger 文档

需要安装 `swag` 工具：

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

生成文档：

```bash
make swagger
```

## 接口鉴权

大部分接口需要 `X-User-ID` Header。

1.  调用 `/auth/login` 获取用户信息（包含 ID）。
2.  后续请求在 Header 中带上 `X-User-ID: <ID>`。
