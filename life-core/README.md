# life-core

基于 [Hollow](https://github.com/vaynedu/hollow) 的 Go Web 项目。

## 环境与创建方式

要求 Go 1.25+、Protocol Buffers 编译器，并安装代码生成插件：

```bash
git clone https://github.com/googleapis/googleapis.git /absolute/path/to/googleapis
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.1
(cd /path/to/protoc-gen-myhttp && go install .)
```

`protoc-gen-myhttp` 必须从已合入本次 HTTP adapter 修复的源码目录安装，不要使用尚未包含修复的 latest 版本。
`google/api/annotations.proto` 来自 googleapis 仓库（也可使用已有安装）；`PROTO_INCLUDE` 必须指向 googleapis 根目录，且 `/absolute/path/to/googleapis/google/api/annotations.proto` 必须存在。

Hollow 通过 Go Module 直接依赖远程仓库 `github.com/vaynedu/hollow`。当前项目固定到包含所需能力的提交 `85dc87f`，不使用本机路径 `replace`，也不复制 Hollow 源码。

## 开发

首次运行先复制示例配置，并填写本地数据库等私有参数：

```bash
cp conf.example.yaml conf.yaml
```

`conf.yaml` 仅保存在本机，不提交到 Git。

项目调用链：

```text
proto/*_myhttp.pb.go（自动生成）
→ control（接口校验、业务编排和响应组装）
→ service（核心业务逻辑）
→ dao（数据访问）
→ model（数据库模型）

control / service → third_party（微信、腾讯云 ASR、MiniMax 等外部服务）
main → infrastructure（MySQL、Redis、定时调度等基础设施）
```

`make proto` 只更新 `proto/*.pb.go` 和 `proto/*_myhttp.pb.go`。`control/`、`service/`、`dao/`、`model/` 是手写代码，不会被覆盖。

目录职责：

```text
third_party/                    外部服务适配器
  wechat/                       微信登录接口
  minimax/                      MiniMax 文本整理和 AI 总结
  tencentasr/                   腾讯云语音识别
infrastructure/                 可替换的基础设施组件
  database/                     MySQL、Redis 连接与生命周期
  scheduler/                    进程内定时调度及任务注册
dao/                            数据库访问实现（即 repository 层）
service/                        业务规则和用例编排
control/                        HTTP/Proto 接口入口与依赖组装
```

`repository` 不作为通用组件目录；如果后续将 `dao` 更名为 `repository`，其中也只放持久化访问代码。未来接入 etcd、消息队列等技术组件时，统一放入 `infrastructure/`。

任务接口按以下职责拆分：

```text
proto/life_core.proto       接口协议与路由
control/task.go             组装依赖并调用 Service
service/task.go             参数校验、业务编排和响应组装
dao/task.go                 GORM 数据访问
model/task.go               数据库模型
```

Control 按 handler 四段式组织：日志、组装 Service、`InitAndCheckParam`、调用业务并返回响应。Service 的数据库和 DAO 均由 Control 显式注入；DAO 使用 `sqlmock` 验证生成的 SQL。带 `integration` 标签的测试会连接 `conf.yaml` 中的 MySQL，完成真实 HTTP 链路验证。

项目默认提供 `GET /v1/life-core/health`。该接口定义在 Proto 中，并经过自动 HTTP Adapter、`control.Health` 和 `service.Health`。

登录与鉴权：

```text
POST /v1/life-core/auth/login    使用 wx.login 返回的 code 登录
POST /v1/life-core/user/current  获取当前用户和白名单状态
POST /v1/life-core/access/status 获取当前用户审批状态
```

登录成功响应中的核心字段：

```json
{
  "access_token": "<JWT>",
  "expires_at": "2026-09-09T22:30:00+08:00",
  "user": {}
}
```

除健康检查和登录接口外，`/v1/life-core/*` 均要求携带
`Authorization: Bearer <access_token>`。登录响应返回 `access_token` 和 `expires_at`，
小程序将它们缓存在本地，后续请求由统一请求层自动添加请求头。Token 使用 JWT，服务端不落库；
鉴权中间件校验签名和过期时间后，根据 Token 中的用户 ID 查询用户并检查账号状态。
Token 默认有效期为 180 天，可通过 `auth.token_ttl` 调整。Token 过期、签名错误或格式非法时
统一返回 HTTP 401。已经登录过的小程序收到 401 后会调用
`wx.login` 静默获取新 code，重新登录并只重试原请求一次；首次使用仍由用户主动点击登录。

鉴权范围由配置控制：`auth.enabled=false` 时关闭 Token 中间件；开启时仅校验
`auth.required_paths` 命中的路径。规则支持完整路径，以及以 `*` 结尾的模块前缀，
例如 `/v1/life-core/task/*`。登录和健康检查默认不在鉴权列表中。

`users.wx_open_id` 保存微信登录换取的 OpenID，并由数据库唯一索引
`uk_users_wx_open_id` 保证唯一。`/v1/life-core/user/current` 允许待审核用户访问，其余业务接口要求用户状态为
`approved`；管理员接口还会额外检查 `role=admin`。

访问审批接口：

```text
POST /v1/life-core/admin/access_request/list    管理员分页查看待审批、已通过或已拒绝用户
POST /v1/life-core/admin/access_request/modify  管理员通过或拒绝普通用户
```

新用户首次登录后自动创建为 `pending`，仍可调用状态接口，但访问任务、生活记录、吃喝和地点接口会返回
HTTP 403。管理员审批为 `approved` 后即可访问业务数据；拒绝后状态为 `rejected`。

本地 `conf.yaml` 默认开启微信 Mock 登录，以便没有 AppSecret 时先跑通完整 JWT 流程。
部署前必须配置真实 `wechat.app_secret`、关闭 `wechat.mock_enabled`，并替换
`auth.jwt_secret`。`bootstrap_first_user` 开启时，第一个登录用户会成为已通过的管理员。

配置仅在启动时从本地 `conf.yaml` 读取。MySQL 和 Redis 默认关闭；需要使用时将对应 `enabled` 改为 `true` 并填写连接信息，启动过程会自动建连，退出时自动关闭。

业务代码可直接使用自动携带 Request ID 的 Logger：

```go
logger := hlog.FromContext(ctx).Named("RestaurantService.List").
	With(hlog.String("open_id", openID))
```

新增单元测试统一使用 GoConvey 的 `Convey`、`So` 风格。

```bash
make proto PROTO_INCLUDE=/absolute/path/to/googleapis
make deps
make test
make integration-test
make migrate
make run
```

定时任务使用进程内 `robfig/cron/v3`，不依赖 Linux cron，因此 Windows、macOS 和 Linux
使用同一套配置。当前提供每日任务总结任务：按 `Asia/Shanghai` 时区每天 06:00 执行，
同一任务不会重叠运行，支持超时、panic 恢复、优雅关闭和启动补跑。启用方式见
`conf.example.yaml` 的 `scheduler` 配置。启用前先执行 `make migrate` 创建
`task_ai_summaries` 表；当天任务事实未变化时会复用数据库结果，不会重复调用 AI。

任务模块接口：

```text
POST /v1/life-core/task/list
POST /v1/life-core/task/detail
POST /v1/life-core/task/create
POST /v1/life-core/task/modify
POST /v1/life-core/task/status/modify
POST /v1/life-core/task/delete
```

生活记录模块接口：

```text
POST /v1/life-core/activity_log/list
POST /v1/life-core/activity_log/detail
POST /v1/life-core/activity_log/create
POST /v1/life-core/activity_log/modify
POST /v1/life-core/activity_log/delete
POST /v1/life-core/activity_log/stats
```

业务接口统一采用 `POST /v1/life-core/{module}/{action}`，请求参数放在 JSON Body 中。
路由与请求、响应结构统一定义在 `proto/life_core.proto`，由 HTTP Adapter 自动生成。

吃喝模块接口：

```text
POST /v1/life-core/restaurant/list
POST /v1/life-core/restaurant/recommend
POST /v1/life-core/restaurant/detail
POST /v1/life-core/restaurant/create
POST /v1/life-core/restaurant/modify
POST /v1/life-core/restaurant/delete
POST /v1/life-core/restaurant/consumption/list
POST /v1/life-core/restaurant/consumption/create
POST /v1/life-core/restaurant/consumption/modify
POST /v1/life-core/restaurant/consumption/delete
```

消费记录的新增、编辑、删除均在事务内重算店铺的消费次数、累计金额和最近一次消费。

地点模块接口：

```text
POST /v1/life-core/place/list
POST /v1/life-core/place/detail
POST /v1/life-core/place/create
POST /v1/life-core/place/modify
POST /v1/life-core/place/delete
```

`status` 支持 `visited`（去过）和 `wishlist`（想去）。经纬度字段当前仅预留，不参与距离计算。
