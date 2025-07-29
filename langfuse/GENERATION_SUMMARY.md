# Langfuse API 客户端生成总结

## 生成过程

使用 `oapi-codegen` 从 OpenAPI 规范文件 `docs/openapi.yml` 生成了完整的 Go 客户端代码。

### 生成命令

```bash
# 生成客户端代码
oapi-codegen -generate client -package client docs/openapi.yml > client/client.go

# 生成类型定义
oapi-codegen -generate types -package client docs/openapi.yml > client/types.go

# 生成服务器代码（可选）
oapi-codegen -generate server -package client docs/openapi.yml > client/server.go
```

## 生成的文件

### 1. client.go (365KB, 12,573 行)
- 主要的客户端代码
- 包含 HTTP 客户端实现
- 所有 API 方法的实现
- 响应解析函数
- 客户端配置选项

### 2. types.go (128KB, 3,614 行)
- 所有 API 相关的类型定义
- 请求和响应结构体
- 枚举类型
- 联合类型（如 CreateScoreValue）
- 类型转换方法

### 3. server.go (可选)
- 服务器端代码
- 如果需要实现服务器端 API

### 4. example.go (4.5KB, 185 行)
- 使用示例代码
- 展示如何创建客户端
- 展示如何设置认证
- 展示如何调用各种 API
- 包含错误处理示例

### 5. client_test.go (2.8KB, 108 行)
- 单元测试
- 验证客户端创建
- 验证类型操作
- 验证配置选项

### 6. README.md (5.2KB, 243 行)
- 详细的使用文档
- 安装说明
- API 使用示例
- 配置选项说明
- 错误处理指南

## 主要功能

### 客户端类型

1. **Client** - 基本客户端
   - 直接返回 HTTP 响应
   - 需要手动解析响应

2. **ClientWithResponses** - 带响应的客户端（推荐）
   - 自动解析响应
   - 类型安全的响应处理
   - 更好的错误处理

### 认证方式

支持基本认证：
```go
clientWithResponses.ClientInterface.(*Client).RequestEditors = append(
    clientWithResponses.ClientInterface.(*Client).RequestEditors,
    func(ctx context.Context, req *http.Request) error {
        req.SetBasicAuth("your-public-key", "your-secret-key")
        return nil
    },
)
```

### 主要 API 功能

1. **健康检查** - `HealthHealthWithResponse`
2. **评分管理** - `ScoreCreateWithResponse`, `ScoreV2GetWithResponse`
3. **Trace 管理** - `TraceListWithResponse`, `TraceGetWithResponse`
4. **观察数据** - `ObservationsGetManyWithResponse`
5. **数据集管理** - `DatasetsListWithResponse`
6. **项目管理** - `ProjectsGetWithResponse`
7. **评论管理** - `CommentsCreateWithResponse`

### 特殊类型处理

#### CreateScoreValue 联合类型
```go
// 数值评分
scoreValue := CreateScoreValue{}
scoreValue.FromCreateScoreValue0(0.95)

// 字符串评分
scoreValue := CreateScoreValue{}
scoreValue.FromCreateScoreValue1("excellent")
```

#### 可选字段
```go
// 使用指针创建可选字段
scoreId := "score-123"
comment := "High accuracy score"
request := CreateScoreRequest{
    Id:      &scoreId,
    Name:    "accuracy",
    Comment: &comment,
}
```

## 依赖

- `github.com/oapi-codegen/runtime` - 运行时支持
- 标准库：`net/http`, `context`, `encoding/json` 等

## 测试

运行测试：
```bash
go test ./client
```

## 重新生成

如果 OpenAPI 规范更新，可以重新运行生成命令来更新客户端代码。

## 注意事项

1. 生成的代码是类型安全的
2. 所有可选字段都使用指针类型
3. 响应处理支持多种状态码
4. 支持自定义 HTTP 客户端和传输层
5. 支持请求编辑器进行认证和自定义请求头 