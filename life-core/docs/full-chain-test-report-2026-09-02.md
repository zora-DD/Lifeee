# Life 小程序全链路测试报告（2026-09-02）

## 1. 测试结论

- 结论：通过。
- 后端实际注册 30 条 `/v1` 接口，路由清单校验全部通过，不存在旧 `/api/*` 路由。
- 登录、当前用户、任务、生活记录、吃喝、消费记录、地点的已实现接口均完成真实 MySQL HTTP 集成测试。
- 小程序当前页面使用的 28 条业务请求路径全部采用 `/v1/*`；29 个前端 JavaScript 文件通过语法检查。
- Go 全量单测、静态检查、构建、真实数据库集成测试均通过；集成测试累计 133 个断言通过。
- 本机 `127.0.0.1:8081` 与局域网 `10.10.151.53:8081` 均可访问，真机当前局域网地址与 `app.js` 配置一致。

## 2. 测试环境

| 项目 | 测试值 |
|---|---|
| 日期 | 2026-09-02 |
| 操作系统 | Windows amd64 |
| Go | go1.25.1 |
| Node.js | v22.22.2 |
| 后端框架 | Hollow + Gin + GORM |
| 服务端口 | 8081 |
| 数据库 | 私有 MySQL `life`，真实读写测试通过 |
| Redis | 未启用 |
| 本地微信登录 | Mock code 换取 OpenID |
| 鉴权 | JWT Bearer Token，默认 180 天 |

报告不记录数据库 DSN、JWT 密钥、微信密钥或完整 Token。

## 3. 页面与业务链路

| 页面/模块 | 已验证链路 | 结果 |
|---|---|---|
| 首次进入/主页 | 未登录不加载业务数据；登录后缓存 Token；获取当前用户；加载当月生活统计 | 通过 |
| 任务 | 列表、分页、筛选、搜索、详情、新建、编辑、完成状态、删除、完成项沉底 | 通过 |
| 生活记录 | 列表、分页、时间筛选、输入类型筛选、新建、编辑、详情、删除、月度统计、年度统计 | 通过 |
| 吃喝 | 列表、分类/偏好/关键词筛选、新建、详情、编辑、随机推荐、删除 | 通过 |
| 消费记录 | 新建、分页列表、编辑、删除；次数、总金额、最近时间及最近消费内容联动更新 | 通过 |
| 地点 | 列表、分类/状态/关键词筛选、新建、详情、编辑、已去/想去数据规范化、删除 | 通过 |
| 我的贡献 | 分别请求任务、吃喝、地点列表，再按当前用户 ID 在前端聚合 | 通过 |
| 我的收藏 | 当前仅占位，不发起后端请求 | 符合当前范围 |
| 账户页 | 退出登录清除头像资料、Token 和过期时间缓存 | 通过 |

## 4. 接口逐项结果

统一成功响应为 HTTP 2xx 且业务 `code=0`。参数错误返回 400，未认证返回 401，无权限返回 403，已删除资源详情返回 404。

| 模块 | 方法与 URL | 验证内容 | 结果 |
|---|---|---|---|
| 健康检查 | `GET /v1/health` | 本机及局域网可达 | 通过 |
| 登录 | `POST /v1/auth/login` | code 登录、用户落库、签发 JWT | 通过 |
| 用户 | `POST /v1/user/current` | Bearer Token 获取当前用户 | 通过 |
| 任务 | `POST /v1/task/list` | 筛选、分页、`has_more`、非法优先级 | 通过 |
| 任务 | `POST /v1/task/detail` | 按 ID 查询、删除后 404 | 通过 |
| 任务 | `POST /v1/task/create` | 标题、描述、优先级、分类、截止时间 | 通过 |
| 任务 | `POST /v1/task/modify` | 编辑字段并回读最新值 | 通过 |
| 任务 | `POST /v1/task/status/modify` | todo/done 状态切换 | 通过 |
| 任务 | `POST /v1/task/delete` | 软删除 | 通过 |
| 生活记录 | `POST /v1/activity_log/list` | 时间范围、输入类型、分页、`has_more` | 通过 |
| 生活记录 | `POST /v1/activity_log/detail` | 按 ID 查询、删除后 404 | 通过 |
| 生活记录 | `POST /v1/activity_log/create` | 文本/语音类型、发生时间、时长、音频 URL | 通过 |
| 生活记录 | `POST /v1/activity_log/modify` | 内容、时间、时长、输入类型更新 | 通过 |
| 生活记录 | `POST /v1/activity_log/delete` | 软删除 | 通过 |
| 生活记录 | `POST /v1/activity_log/stats` | 月度/年度记录数、活跃天数、总时长、日/月分布 | 通过 |
| 吃喝 | `POST /v1/restaurant/list` | 分类、偏好、关键词、分页、`has_more` | 通过 |
| 吃喝 | `POST /v1/restaurant/detail` | 店铺详情与消费汇总 | 通过 |
| 吃喝 | `POST /v1/restaurant/create` | 店铺字段及封面 URL 字段 | 通过 |
| 吃喝 | `POST /v1/restaurant/modify` | 店铺字段编辑 | 通过 |
| 吃喝 | `POST /v1/restaurant/delete` | 软删除、删除后 404 | 通过 |
| 吃喝 | `POST /v1/restaurant/recommend` | 按分类随机推荐 | 通过 |
| 消费 | `POST /v1/restaurant/consumption/list` | 分页、总数、`has_more` | 通过 |
| 消费 | `POST /v1/restaurant/consumption/create` | 日期、金额、吃喝内容、备注 | 通过 |
| 消费 | `POST /v1/restaurant/consumption/modify` | 消费记录编辑及店铺汇总重算 | 通过 |
| 消费 | `POST /v1/restaurant/consumption/delete` | 删除及次数/总金额重算 | 通过 |
| 地点 | `POST /v1/place/list` | 分类、去过/想去、关键词、分页、`has_more` | 通过 |
| 地点 | `POST /v1/place/detail` | 按 ID 查询、删除后 404 | 通过 |
| 地点 | `POST /v1/place/create` | 地点、特色、次数、时间、坐标和封面 URL 字段 | 通过 |
| 地点 | `POST /v1/place/modify` | 编辑；改为未去过时次数和最近时间归零 | 通过 |
| 地点 | `POST /v1/place/delete` | 软删除 | 通过 |

## 5. 关键数据一致性验证

- 任务分页响应同时返回 `page`、`pageSize`、`total`、`has_more`。
- 任务数据库查询排序为：未完成优先、已完成沉底；同状态按 `created_at DESC, id DESC`。
- 消费新增两笔后，店铺详情中的消费次数和总金额正确累计；编辑、删除消费后重新计算正确。
- 地点切换为“想去”时，后端将 `visitCount` 归零、`lastVisitedAt` 清空，避免脏状态。
- 生活记录月度统计正确计算记录数、活跃天数、总时长、平均时长、最长时长和每日分布；年度统计返回完整十二个月。
- 小程序 lowerCamelCase 请求字段（如 `pageSize`、`dueAt`、`restaurantId`）与 snake_case 请求均可解析。

## 6. 鉴权与会话测试

| 场景 | 预期 | 结果 |
|---|---|---|
| 无 Token 访问受保护业务接口 | 401 | 通过 |
| 伪造 Token 访问受保护业务接口 | 401 | 通过 |
| 过期 Token | 401 | 通过 |
| 已通过用户访问业务接口 | 正常放行 | 通过 |
| 待审核用户访问业务接口 | 403，不触发重新登录 | 通过 |
| 前端收到 401 | 调用一次 `wx.login`，获取新 Token 后原请求重试一次 | 通过 |
| 前端收到 403 | 直接展示无权限，不重新登录，保留会话 | 通过 |
| 多请求同时 401 | 复用同一个续登 Promise，只调用一次 `wx.login` | 通过 |
| JWT 有效期 | 180 天 | 通过 |
| 用户 OpenID | 数据库存储且存在唯一索引 `uk_users_wx_open_id` | 通过 |

## 7. 自动化测试结果

执行命令：

```text
go build ./...
go test ./...
go vet ./...
go test -tags=integration ./router -v -count=1
node --check <全部小程序 JavaScript 文件>
node --test tests/*.test.js
```

结果：

- Go 构建：通过。
- Go 全量单测：通过；源码中共 45 个 `Test*` 测试函数。
- Go 静态检查：通过。
- 真实 MySQL HTTP 集成测试：6 个测试入口、133 个断言全部通过。
- 小程序 JavaScript：29 个文件语法检查通过。
- 小程序新增自动化测试：6/6 通过，覆盖 API 路径契约、Token Header、401 自动续登、并发 401 复用续登、403 不续登。
- 普通单测覆盖率：`auth 57.6%`、`config 65.2%`、`control 12.7%`、`dao 24.9%`、`model 100%`、`proto 4.2%`、`router 56.2%`、`service 65.0%`、`wechat 63.6%`。生成代码和启动装配包拉低了整体数字，核心业务另由真实数据库集成测试覆盖。

## 8. 本轮补齐内容

- 新增生活记录 HTTP 集成测试，覆盖 CRUD、分页、非法筛选、月度统计和年度统计。
- 新增 30 条注册路由精确校验，防止 `/api/*` 旧路径回归。
- 补齐生活记录服务层创建、编辑、删除、字段校验和年度统计单测。
- 新增小程序请求层测试，覆盖 Bearer Header、空参数清理、401 续登和 403 行为。
- 新增小程序 API 路径契约测试，明确已实现接口与预留接口边界。

## 9. 当前未实现的预留能力

以下路径存在于前端规划或技术方案中，但后端尚未注册，因此不计入本次“30 条已实现接口通过率”：

- `POST /v1/user/contribution/list`：当前“我的贡献”由前端并行读取三个列表后聚合。
- 收藏相关接口：当前收藏页仅占位。
- `POST /v1/access/apply`
- `POST /v1/access/status`
- `POST /v1/admin/access_request/list`
- `POST /v1/admin/access_request/modify`
- `POST /v1/restaurant/cover/upload`
- `POST /v1/place/cover/upload`
- 生活记录语音上传/转写接口：目前只保存预留 `audioUrl`。

## 10. 上线前风险与建议

1. 当前本地配置启用微信 Mock 登录。生产必须配置真实小程序 AppID/AppSecret、关闭 Mock，并实测微信 `code2session`。
2. 当前小程序请求地址是局域网 HTTP 地址。发布前应切换为 `https://life-core.cn`，配置微信合法 request/uploadFile 域名，并增加明确的 test/prod 环境配置。
3. 封面上传入口后端尚未实现。上传按钮正式开放前，应先实现对象存储上传接口；数据库仍只保存 URL/ObjectKey。
4. 当前业务内容对所有已通过用户可见；编辑和删除是否仅允许创建者操作尚未形成明确规则。上线多人使用前建议确定权限策略并增加越权测试。
5. 现有技术文档中的 `/api/*`、Redis 随机 Token、7 天有效期、pending 用户访问等描述已经过时。当前真实实现以 `/v1/*`、无 Redis、JWT 180 天、401/403 分流为准。

## 11. 最终判定

当前已实现范围可以完成“小程序登录 → 获取 JWT → 访问任务/生活记录/吃喝/地点 → 创建/查询/编辑/筛选分页/删除 → 主页生活回顾”的完整闭环。自动化与真实数据库测试均通过，可进入真实微信登录、HTTPS 域名和上传能力的下一阶段联调。
