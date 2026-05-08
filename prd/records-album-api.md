# Records Album 联调契约

适用范围：`/api/records/albums` 相关接口。

## 变更结论

- records 中的图片内容现在以 `album` 为一等资源，不再以单张 `photo` 作为 records 列表项。
- 一个 `album` 可以包含多张 `photos`。
- 相册响应里的 `photos` 只返回展示所需字段，不再返回单张照片的 `description`、`source`。
- 相册响应里的 `likes` 返回点赞数，不再返回点赞数组。
- 旧路由 `/api/records/photos` 已切换为 `/api/records/albums`，前端应全部改用新路由。

## 认证

所有接口都需要：

```http
Authorization: Bearer <access-token>
Content-Type: application/json
```

## 数据结构

### AlbumPhoto

请求体结构：

```json
{
  "url": "https://example.com/1.jpg"
}
```

字段说明：

- `url`: 照片地址，必填

### AlbumPhotoResponse

响应体结构：

```json
{
  "id": 101,
  "album_id": 12,
  "group_id": 3,
  "uploader_id": 8,
  "url": "https://example.com/1.jpg"
}
```

### Album

```json
{
  "id": 12,
  "group_id": 3,
  "creator_id": 8,
  "title": "五一出游",
  "description": "西湖和周边照片",
  "visibility": "public",
  "likes": 2,
  "photos": [
    {
      "id": 101,
      "album_id": 12,
      "group_id": 3,
      "uploader_id": 8,
      "url": "https://example.com/1.jpg"
    }
  ]
}
```

说明：

- `visibility` 目前支持 `public` 和 `private`
- `photos` 会随 album 一起返回，前端无需再单独请求子资源
- 后端还会返回通用时间字段，联调时可按实际响应读取

## 接口清单

### 1. 创建相册

`POST /api/records/albums`

请求示例：

```json
{
  "title": "五一出游",
  "description": "西湖和周边照片",
  "visibility": "public",
  "photos": [
    {
      "url": "https://example.com/1.jpg",
      "url": "https://example.com/1.jpg"
    },
    {
      "url": "https://example.com/2.jpg"
    }
  ]
}
```

约束：

- `title` 必填，最大 255 字符
- `description` 选填，最大 500 字符
- `photos` 必填，且至少 1 张

成功响应示例：

```json
{
  "code": "SUCCESS",
  "message": "创建相册成功",
  "data": {
    "id": 12,
    "group_id": 3,
    "creator_id": 8,
    "title": "五一出游",
    "description": "西湖和周边照片",
    "visibility": "public",
    "likes": 0,
    "photos": [
      {
        "id": 101,
        "album_id": 12,
        "group_id": 3,
        "uploader_id": 8,
        "url": "https://example.com/1.jpg"
      }
    ]
  }
}
```

### 2. 更新相册

`PUT /api/records/albums/:id`

请求体结构与创建接口一致。

当前后端语义：

- 更新相册时，`photos` 按整组替换处理
- 前端提交的 `photos` 应视为“更新后的完整照片列表”，不是增量 patch

成功响应：

- `code = SUCCESS`
- `message = 更新相册成功`
- `data` 为更新后的完整 album 对象

### 3. 获取相册列表

`GET /api/records/albums?page=1&page_size=20`

成功响应示例：

```json
{
  "code": "SUCCESS",
  "message": "获取相册列表成功",
  "data": [
    {
      "id": 12,
      "group_id": 3,
      "creator_id": 8,
      "title": "五一出游",
      "description": "西湖和周边照片",
      "visibility": "public",
      "likes": 0,
      "photos": [
        {
          "id": 101,
          "album_id": 12,
          "group_id": 3,
          "uploader_id": 8,
          "url": "https://example.com/1.jpg"
        }
      ]
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

### 4. 删除相册

`DELETE /api/records/albums/:id`

成功响应示例：

```json
{
  "code": "SUCCESS",
  "message": "删除相册成功"
}
```

## 常见错误

- `400 BAD_REQUEST`: 相册缺少照片，或请求字段校验失败
- `403 FORBIDDEN`: 当前用户无权操作该相册
- `404 RESOURCE_NOT_FOUND`: 相册不存在

## 前端迁移建议

- 原先依赖 `/records/photos` 的页面，应改为渲染相册列表
- 进入 records 列表后，列表项应以 album 为单位展示
- 如果前端存在编辑相册能力，提交时请带完整 `photos` 数组
