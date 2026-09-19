# Echo Layout

[![Go](https://github.com/bogeia/echo-layout/actions/workflows/go.yml/badge.svg)](https://github.com/bogeia/echo-layout/actions/workflows/go.yml)

参考 [kratos-layout](https://github.com/go-kratos/kratos-layout) 分层构建的纯 HTTP Web 服务基础框架，不含 gRPC 与 Protobuf。

| 用途 | 选型 |
| --- | --- |
| HTTP 路由 | [Echo v5](https://github.com/labstack/echo) |
| 接口定义、校验、OpenAPI | [Huma v2](https://github.com/danielgtaylor/huma) |
| 依赖注入 | [Wire](https://github.com/google/wire) |
| 配置 | [Viper](https://github.com/spf13/viper)（仅 YAML） |
| 日志 | `log/slog` + [lumberjack](https://github.com/natefinch/lumberjack) |

## 快速开始

```sh
go run ./cmd/server --conf configs/config.yaml
```

启动后访问 <http://127.0.0.1:8080/docs>，可在页面中直接选择接口、填写参数并发送请求。

编译后启动：

```sh
make build
./bin/server --conf configs/config.yaml
```

Docker 启动：

```sh
docker build -t echo-layout .
docker run -p 8080:8080 -v /path/to/conf:/data/conf echo-layout
```

镜像不包含配置文件，需挂载一个包含 `config.yaml` 的目录到 `/data/conf`。容器内 `server.address` 必须为 `:8080` 或 `0.0.0.0:8080`，否则容器外无法访问。

## 目录结构

```text
.
├── cmd/server
│   ├── main.go          # 命令行参数、加载配置、初始化日志、处理退出信号
│   ├── wire.go          # Wire 注入声明（wireinject 构建标签，不参与编译）
│   └── wire_gen.go      # Wire 生成的组装代码，勿手动修改
├── configs/config.yaml  # 示例配置
├── internal
│   ├── server           # HTTP 服务：Echo 实例、中间件、路由注册、启动与优雅关闭
│   ├── service          # 接口层：请求与响应结构、参数转换、错误映射
│   ├── biz              # 业务层：领域模型、业务规则、仓储接口
│   ├── data             # 数据层：存储客户端与仓储实现
│   ├── config           # 配置加载与校验
│   ├── middleware/logging # 入参日志包装器（可选，默认未启用）
│   └── infra/logging    # slog 日志与文件轮转
├── Dockerfile
└── Makefile
```

## 分层

请求调用链：

```text
HTTP → server（路由） → service → biz → data（通过 biz 定义的仓储接口）
```

源码依赖方向：

```text
server → service → biz ← data
```

- **service** 负责 HTTP 与业务之间的转换：定义请求、响应结构和校验规则，把 biz 错误映射为 HTTP 错误。
- **biz** 是核心，只依赖标准库和 Wire（用于 `ProviderSet`）。它定义领域模型（如 `User`）和仓储接口（如 `UserRepo`），不依赖 data。单元测试可以直接使用测试仓储。
- **data** 实现 biz 的仓储接口。`data.go` 中的 `Data` 持有各仓储共享的长期存储客户端（如 `*sql.DB`、Redis），由 `NewData` 创建并返回 cleanup。当前 `Data` 为空实现，关闭时打印一行日志；`user.go` 中的仓储只打印日志，不保存数据。
- **server** 创建 Echo 实例并注册中间件，在 `routes.go` 的 `NewRouter` 中集中注册所有路由。

## 依赖注入

每层在自己的包里通过 `ProviderSet` 暴露构造函数：

| 文件 | ProviderSet |
| --- | --- |
| `internal/data/data.go` | `NewData`、`NewUsers` |
| `internal/biz/biz.go` | `NewUsers` |
| `internal/service/service.go` | `NewUsers` |
| `internal/server/server.go` | `NewRouter` |

`cmd/server/wire.go` 声明注入函数 `wireRouter(*slog.Logger) (*echo.Echo, func(), error)`，由 Wire 生成 `wire_gen.go`。构造函数返回 `error` 时，Wire 自动传递错误；返回 cleanup 时，Wire 按创建的逆序串联。`main` 只负责调用 `wireRouter`，并在退出时执行 cleanup。

Wire 版本通过 `go.mod` 中的 `tool` 指令固定；`make generate` 和 `make init` 使用相同版本。

## 新增业务模块

以新增 `Order` 为例：

1. **biz**：新建 `internal/biz/order.go`，定义模型、`OrderRepository` 接口、`NewOrders` 和业务方法。把 `NewOrders` 加入 `biz.ProviderSet`。
2. **data**：新建 `internal/data/order.go`，实现仓储。构造函数签名为 `NewOrders(data *Data, log *slog.Logger) biz.OrderRepository`，返回 biz 接口，无需 `wire.Bind`。把它加入 `data.ProviderSet`。
3. **service**：新建 `internal/service/order.go`，定义输入、输出结构和处理函数。需要记录日志时给 `NewOrders` 显式注入 `*slog.Logger`。把 `NewOrders` 加入 `service.ProviderSet`。
4. **server**：`NewRouter` 增加 `orders *service.Orders` 参数，并用 `huma.Register` 注册路由。
5. 执行 `make generate`，重新生成 `wire_gen.go`。

构造函数名在各层包内可以重复，Wire 按类型区分。

接入数据库时：在 `Data` 中添加客户端字段，在 `NewData` 中创建客户端，并在 cleanup 中关闭。若需要数据库配置，在 `internal/config` 中新增配置段，把配置作为 `NewData` 的参数，再给 `wireRouter` 增加对应参数。

## 配置

启动必须通过 `--conf` 指定 YAML 文件，且程序定义的配置字段全部必填。未知字段会被忽略；必填字段缺失、类型错误、值非法或文件读取失败时，程序终止启动。配置不读取环境变量，代码也不回填默认值。

```yaml
server:
  address: 127.0.0.1:8080
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 60s
  shutdown_timeout: 10s

log:
  level: info
  console: false
  filename: logs/app.log
  max_size: 500
  max_backups: 7
  max_age: 30
  compress: true
```

| 字段 | 说明 |
| --- | --- |
| `server.address` | 监听地址，格式为 `host:port`。host 可以是 IP、主机名或留空，如 `127.0.0.1:8080`、`localhost:8080`、`:8080`、`'[::1]:8080'`。端口范围 1–65535 |
| `server.read_timeout` | 读取请求的超时时间，同时用作读取请求头的超时时间 |
| `server.write_timeout` | 写响应的超时时间 |
| `server.idle_timeout` | Keep-Alive 空闲连接的超时时间 |
| `server.shutdown_timeout` | 收到 SIGINT/SIGTERM 后等待请求处理完成的最长时间，超时后强制关闭 |
| `log.level` | `debug`、`info`、`warn`、`error`，同时作用于控制台和文件 |
| `log.console` | 是否向控制台输出 text 格式日志；文件日志始终启用 |
| `log.filename` | 日志文件路径，相对于进程工作目录；目录不存在时自动创建 |
| `log.max_size` | 单个日志文件的最大大小，单位 MB，必须大于 0 |
| `log.max_backups` | 保留的旧日志文件数量；`0` 表示不按数量删除 |
| `log.max_age` | 旧日志保留天数；`0` 表示不按时间删除 |
| `log.compress` | 是否用 gzip 压缩轮转后的旧日志；当前日志文件不压缩 |

所有时长字段必须是带单位的字符串，如 `10s`、`1m`，且必须大于 0。

## 日志

- `log.console: true` 时控制台输出 text 格式；文件始终输出 JSON 格式。
- 应用各层通过构造函数注入 logger。启动时同时调用 `slog.SetDefault`，让第三方库使用默认 logger 输出的日志也写入控制台和文件。
- 访问日志的消息为 `http request`，字段包括 `method`、`path`、`status`、`latency`。状态码大于等于 500 或处理出错时，日志级别为 `ERROR`；Echo 返回错误时附带 `error` 字段。Huma 内部错误由 service 单独记录。

### 入参日志（可选）

`internal/middleware/logging` 提供一个 Huma handler 包装器，用于记录业务入参。**默认不启用**，演示路由均未包装。需要时在注册路由时逐个包装：

```go
operation := huma.Operation{OperationID: "create-user", Method: http.MethodPost, Path: "/users", DefaultStatus: http.StatusCreated}
huma.Register(api, operation, logging.Handler(log, operation.OperationID, users.Create))
```

包装后，在 Huma 解析和校验成功、调用 service 之前，输出一条 DEBUG 级别的 `request args`，包含 `operation` 和结构化 `args`（请求类型中声明的 path、query、body 等字段）：

```json
{"level":"DEBUG","msg":"request args","operation":"create-user","args":{"Body":{"name":"Alice"}}}
```

由于是 DEBUG 级别，`log.level: info` 时不输出，排查问题时把级别调成 `debug` 即可。JSON 解析或 Huma 校验失败时不会输出（handler 未被调用），service 内的业务校验失败仍会输出。包装器不读取原始请求体，也不记录响应体。

请求类型包含密码、令牌等敏感字段时，应在该输入类型上实现标准库 `slog.LogValuer`，返回脱敏后的日志值；处理函数仍收到原始输入。未实现时记录完整的已解析结构体，不自动脱敏。

## HTTP 约定

- **panic 恢复**：handler 中的 panic 会被恢复，并返回 500。
- **错误格式**：所有错误响应统一为 Huma `ErrorModel`，`Content-Type` 为 `application/problem+json`。包括 Huma 的校验错误、Echo 的 404 和 405，以及未处理的错误。
- **内部错误**：500 响应不暴露内部错误详情。service 层把未知错误转换为 500 之前，会先用 `slog` 记录原始错误。
- **API 文档**：Huma 自动提供 `/docs`（文档页面，使用外部 CDN 资源）、`/openapi.json` 和 `/openapi.yaml`，接口修改后重启服务即可更新。
- **CORS**：未启用，需要时在 `internal/server/routes.go` 的 `withMiddleware(...)` 中添加 `middleware.CORS()`。

## 演示接口

`/users` 用于演示完整的调用链。data 层不保存数据，因此创建后再查询仍返回 404。

| 方法 | 路径 | 成功响应 |
| --- | --- | --- |
| POST | `/users` | 201，返回生成的 `id` 和 `name` |
| GET | `/users` | 200，空数组 `[]` |
| GET | `/users/{id}` | 404 |
| PUT | `/users/{id}` | 200，返回请求中的 `id` 和 `name` |
| DELETE | `/users/{id}` | 204 |

请求体示例为 `{"name":"Alice"}`。`name` 必填，长度 1–100，不能只包含空白字符；`id` 长度 1–128。JSON 格式错误返回 400，校验失败返回 422。

```sh
curl -i -X POST http://127.0.0.1:8080/users \
  -H 'Content-Type: application/json' -d '{"name":"Alice"}'
```

## 开发

| 命令 | 说明 |
| --- | --- |
| `make build` | 编译到 `bin/` |
| `make test` | 运行测试 |
| `make generate` | 执行 `go generate`，重新生成 `wire_gen.go` |
| `make init` | 安装 `wire` 命令行工具（可选，`make generate` 不依赖它） |

GitHub Actions 会依次检查：生成代码是否最新、`gofmt`、`go vet`、`go test -race`。

## License

[MIT](LICENSE)
