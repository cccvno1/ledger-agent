# Ledger Agent 智能记账助手

一个基于大语言模型的**自然语言记账 Agent**，专为小商贩设计。用户通过微信或 HTTP API 用日常口语记录出货、查账、收款，系统自动理解并结构化存储。


## 技术栈

| 组件 | 选型 |
|------|------|
| 语言 | Go 1.26 |
| LLM | MiniMax-M2.7 (OpenAI-compatible API) |
| Agent 框架 | [CloudWeGo eino](https://github.com/cloudwego/eino) (ReAct Agent) |
| 数据库 | PostgreSQL 16 |
| 微信 | iLink Bot API |
| 部署 | Docker Compose |

## 快速开始

### 前置要求

- Go 1.26+
- Docker & Docker Compose
- [MiniMax API Key](https://platform.minimaxi.com/) (注册后获取)

### 1. 克隆项目

```bash
git clone https://github.com/cccvno1/ledger-agent.git
cd ledger-agent
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env，填入你的 API Key
```

`.env` 文件内容：

```bash
DATABASE_URL=postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable
MINIMAX_API_KEY=your-minimax-api-key-here
AUTH_TOKEN=$(openssl rand -hex 32)   # 生成随机密钥，前端请求时携带
```

### 3. 启动基础设施

```bash
make docker-up          # 启动 PostgreSQL
docker compose ps       # 确认服务健康
```

### 4. 运行数据库迁移

```bash
make migrate
```

### 5. 启动服务

```bash
source .env
APP_ENV=local make run
```

服务启动后监听 `http://localhost:8080`。

### 6. 测试对话

```bash
# 记一笔账
curl -s http://localhost:8080/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"session_id":"demo","message":"张三买了10斤苹果，单价5块"}' | python3 -m json.tool

# 确认保存
curl -s http://localhost:8080/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"session_id":"demo","message":"确认"}' | python3 -m json.tool

# 查账
curl -s http://localhost:8080/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"session_id":"demo","message":"查一下张三的账"}' | python3 -m json.tool
```

## 微信接入（可选）

```bash
# 首次登录：扫码绑定微信
go run ./cmd/wechat-login

# 登录成功后重启服务，微信消息自动桥接到 Agent
APP_ENV=local make run
```

> 凭证保存在本地 `wechat-login` 文件中，下次启动自动加载，无需重复扫码。

## API 接口

完整接口文档见 [docs/API.md](docs/API.md)。

所有接口（除 `/health` 和 `/api/v1/wechat/qrcode*`）需携带认证头：
```
Authorization: Bearer <AUTH_TOKEN>
```

### 核心接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/chat` | 自然语言记账 |
| `GET` | `/api/v1/dashboard` | 首页概览 |
| `GET/POST` | `/api/v1/entries` | 账目列表 / 创建 |
| `PUT/DELETE` | `/api/v1/entries/{id}` | 更新 / 删除账目 |
| `GET` | `/api/v1/customers` | 客户列表 |
| `GET` | `/api/v1/customers/{id}/summary` | 客户汇总 |
| `POST` | `/api/v1/customers/{id}/settle` | 一键结算 |
| `GET/POST` | `/api/v1/payments` | 收款记录 |
| `GET` | `/api/v1/products` | 商品列表 |
| `GET` | `/health` | 健康检查 |

## 对话示例

```
用户: 张三买了10斤苹果，单价5块
助手: 已添加到草稿：张三 苹果 10斤 ×5.00 = 50.00元。确认保存吗？

用户: 确认
助手: ✅ 已保存！

用户: 李四昨天买了5箱橘子单价30，今天买了20斤香蕉单价3块
助手: 已添加 2 条记录到草稿：
      1. 李四 橘子 5箱 ×30.00 = 150.00元 (04-20)
      2. 李四 香蕉 20斤 ×3.00 = 60.00元 (04-21)
      确认保存吗？

用户: 单价改成45       (修改草稿)
用户: 算了不要了       (清空草稿)
用户: 查一下张三的账    (查询)
用户: 张三付了200块     (收款)
```

## 项目结构

```
cmd/server/              应用入口
internal/
  base/
    boot/                依赖组装 + 生命周期
    conf/                配置结构
    middleware/           HTTP 中间件 (RequestID, Logging, Recover)
    router/              路由注册
  domain/                共享业务类型 + 错误码
  chat/                  聊天 Agent（核心）
    agent.go             系统提示词 + Agent 构建
    tools.go             工具实现（记账/查账/收款等）
    service.go           对话服务（消息管理/上下文注入）
    session.go           会话存储（内存/自动过期）
    handler.go           HTTP 处理
    model.go             会话模型（草稿/操作日志）
  customer/              客户管理（CRUD + 搜索）
  product/               商品管理（CRUD + 搜索 + FindOrCreate）
  ledger/                账本（出货记录读写）
  payment/               收款记录
  wechat/                微信消息桥接
configs/                 YAML 配置（base + 环境覆盖）
migrations/postgres/     数据库迁移脚本
docs/                    架构文档 + 测试报告
```

## 架构设计

- **分层架构**：严格的包依赖规则，由 `architecture_test.go` 自动检测
- **ReAct Agent**：LLM 通过工具调用与数据库交互，非直接生成 SQL
- **草稿模式**：所有写操作先进草稿，用户确认后才持久化
- **三层上下文**：操作日志（结构化事实）+ 草稿快照 + 滑动窗口消息

详见 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)。

## 开发命令

```bash
make build       # 编译
make run         # 本地运行
make test        # 运行测试
make vet         # 静态分析
make tidy        # 整理依赖
make docker-up   # 启动 PostgreSQL
make docker-down # 停止 PostgreSQL
make migrate     # 运行数据库迁移
```

## License

MIT
