# Langfuse API 客户端

这个目录包含了使用 `oapi-codegen` 从 OpenAPI 规范生成的 Go 客户端代码。

## 文件说明

- `client.go` - 主要的客户端代码，包含 HTTP 客户端和所有 API 方法
- `types.go` - 所有 API 相关的类型定义
- `server.go` - 服务器端代码（如果需要实现服务器）
- `example.go` - 使用示例
- `README.md` - 本文件

## 安装依赖

```bash
go mod tidy
```

## 基本用法

### 1. 创建客户端

```go
import "your-project/client"

// 创建基本客户端
client, err := client.NewClient("https://cloud.langfuse.com")
if err != nil {
    log.Fatal(err)
}

// 创建带响应的客户端（推荐）
clientWithResponses, err := client.NewClientWithResponses("https://cloud.langfuse.com")
if err != nil {
    log.Fatal(err)
}
```

### 2. 设置认证

```go
// 设置基本认证
clientWithResponses.ClientInterface.(*client.Client).RequestEditors = append(
    clientWithResponses.ClientInterface.(*client.Client).RequestEditors,
    func(ctx context.Context, req *http.Request) error {
        req.SetBasicAuth("your-public-key", "your-secret-key")
        return nil
    },
)
```

### 3. 使用 API

#### 健康检查

```go
ctx := context.Background()
resp, err := clientWithResponses.HealthHealthWithResponse(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
    return
}

if resp.JSON200 != nil {
    fmt.Printf("Health status: %+v\n", resp.JSON200)
}
```

#### 创建评分

```go
// 创建评分值
scoreValue := client.CreateScoreValue{}
scoreValue.FromCreateScoreValue0(0.95) // 数值评分

// 创建评分请求
scoreId := "score-123"
comment := "High accuracy score"
scoreRequest := client.CreateScoreRequest{
    Id:      &scoreId,
    Name:    "accuracy",
    Value:   scoreValue,
    Comment: &comment,
}

// 发送请求
resp, err := clientWithResponses.ScoreCreateWithResponse(ctx, scoreRequest)
if err != nil {
    log.Printf("Error creating score: %v", err)
    return
}

if resp.JSON200 != nil {
    fmt.Printf("Score created: %+v\n", resp.JSON200)
}
```

#### 获取 Traces

```go
limit := 10
page := 1
resp, err := clientWithResponses.TraceListWithResponse(ctx, &client.TraceListParams{
    Limit: &limit,
    Page:  &page,
})
if err != nil {
    log.Printf("Error getting traces: %v", err)
    return
}

if resp.JSON200 != nil {
    fmt.Printf("Traces count: %d\n", len(resp.JSON200.Data))
    for _, trace := range resp.JSON200.Data {
        fmt.Printf("Trace: ID=%s, Name=%s\n", trace.Id, trace.Name)
    }
}
```

#### 获取评分列表

```go
limit := 10
page := 1
resp, err := clientWithResponses.ScoreV2GetWithResponse(ctx, &client.ScoreV2GetParams{
    Limit: &limit,
    Page:  &page,
})
if err != nil {
    log.Printf("Error getting scores: %v", err)
    return
}

if resp.JSON200 != nil {
    fmt.Printf("Scores count: %d\n", len(resp.JSON200.Data))
}
```

## 自定义配置

### 自定义 HTTP 客户端

```go
// 创建自定义传输层
transport := &http.Transport{
    MaxIdleConns:        10,
    IdleConnTimeout:     30 * time.Second,
    DisableCompression:  true,
}

// 创建 HTTP 客户端
httpClient := &http.Client{
    Transport: transport,
    Timeout:   30 * time.Second,
}

// 使用自定义客户端创建 API 客户端
clientWithResponses, err := client.NewClientWithResponses(
    "https://cloud.langfuse.com",
    client.WithHTTPClient(httpClient),
)
```

### 自定义请求编辑器

```go
// 添加自定义请求头
clientWithResponses.ClientInterface.(*client.Client).RequestEditors = append(
    clientWithResponses.ClientInterface.(*client.Client).RequestEditors,
    func(ctx context.Context, req *http.Request) error {
        req.Header.Set("X-Custom-Header", "custom-value")
        return nil
    },
)
```

## 错误处理

```go
resp, err := clientWithResponses.SomeAPIWithResponse(ctx, params)
if err != nil {
    // 网络错误或其他错误
    log.Printf("Request failed: %v", err)
    return
}

if resp.JSON200 != nil {
    // 成功响应
    fmt.Printf("Success: %+v\n", resp.JSON200)
} else {
    // API 错误响应
    fmt.Printf("API Error: %d - %s\n", resp.HTTPResponse.StatusCode, resp.Body)
}
```

## 类型说明

### CreateScoreValue

`CreateScoreValue` 是一个联合类型，可以包含数值或字符串：

```go
// 数值评分
scoreValue := client.CreateScoreValue{}
scoreValue.FromCreateScoreValue0(0.95)

// 字符串评分
scoreValue := client.CreateScoreValue{}
scoreValue.FromCreateScoreValue1("excellent")
```

### 可选字段

使用 `runtime.String()`, `runtime.Int()` 等函数来创建可选字段：

```go
optionalField := "optional"
optionalInt := 42
request := client.SomeRequest{
    RequiredField: "required",
    OptionalField: &optionalField,
    OptionalInt:   &optionalInt,
}
```

## 重新生成代码

如果需要重新生成客户端代码，请运行：

```bash
# 生成客户端代码
oapi-codegen -generate client -package client docs/openapi.yml > client/client.go

# 生成类型定义
oapi-codegen -generate types -package client docs/openapi.yml > client/types.go

# 生成服务器代码（如果需要）
oapi-codegen -generate server -package client docs/openapi.yml > client/server.go
```

## 更多示例

查看 `example.go` 文件获取更多使用示例。 