# Lifeee

Lifeee 是一个由微信小程序和 Go 后端组成的个人生活管理项目，用于管理任务、生活记录、吃喝消费、地点和记忆等内容。

## 项目结构

```text
.
├── .vscode/       VS Code 工作区配置
├── life-core/     Go 后端 API、业务逻辑和数据库迁移
└── life-miniapp/  微信小程序前端
```

后端和小程序分别是独立部分。根目录没有上传 `go.work`，因此运行 Go 命令时请先进入 `life-core` 目录。

## 功能概览

- 任务：创建、编辑、查询和管理任务。
- 生活记录：记录日常活动并查看统计。
- 吃喝：管理餐馆、零食和消费记录。
- 地点：管理去过的地点和想去清单。
- 记忆：整理生活记忆。
- 微信登录与访问审批：后端提供登录、用户状态和访问审批相关接口。

## 技术组成

- 后端：Go、Gin、Hollow、Protocol Buffers、MySQL、Redis（可选）。
- 前端：微信小程序原生框架。
- 后端接口路由和消息结构定义在 `life-core/proto/`，按模块组织的业务代码位于 `control/`、`service/`、`dao/` 和 `model/`。

## 本地开发

### 运行后端

需要 Go 1.25 或更高版本。进入后端目录，复制示例配置，然后按本机环境填写数据库、微信小程序等配置：

```bash
cd life-core
cp conf.example.yaml conf.yaml
```

启动后端：

```bash
make run
```

默认监听地址为 `http://127.0.0.1:8081`。可用健康检查接口确认服务是否启动：

```bash
curl -i http://127.0.0.1:8081/v1/life-core/health
```

### 运行微信小程序

1. 安装并打开微信开发者工具。
2. 在开发者工具中导入仓库里的 `life-miniapp/` 目录。
3. 确认开发者工具使用的 AppID 与项目配置一致。
4. 本地开发时启动后端，并在开发者工具中运行小程序。


## 配置与安全

- 不要提交 `life-core/conf.yaml`、微信 AppSecret、数据库密码、JWT 密钥、云服务密钥、TLS 私钥或其他凭据。
- 后端提供的 `conf.example.yaml` 是配置模板；部署时应使用独立的生产配置，并替换模板中的示例值。
- 微信小程序前端代码会随小程序发布，不能存放任何服务端密钥。

## 许可证

后端许可证见 [`life-core/LICENSE`](life-core/LICENSE)。
