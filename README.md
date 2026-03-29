# hdu-openclaw

杭电龙虾的 Go 版最小可用后端。当前版本已经具备两条基础能力：

- 飞书机器人接收文本消息
- 调用 OpenAI-compatible 大模型 API
- 将回复发回飞书
- 识别并创建一次性 / 每日提醒任务
- 定时扫描 PostgreSQL 并主动发送提醒

## 代码结构

```text
cmd/server
  服务入口，启动 HTTP 服务、数据库连接和 reminder scheduler

internal/bot
  消息编排层，先尝试 reminder，再回落普通聊天

internal/chat
  普通聊天能力和内存上下文

internal/config
  环境变量与 .env 配置加载

internal/feishu
  飞书 webhook 和发送消息能力

internal/llm
  OpenAI-compatible 大模型客户端

internal/reminder
  提醒解析、数据库读写、提醒调度

internal/store
  PostgreSQL 连接初始化
```

## 启动方式

### 1. 启动 PostgreSQL（Docker）

```powershell
docker run -d `
  --name hdu-openclaw-postgres `
  -e POSTGRES_USER=postgres `
  -e POSTGRES_PASSWORD=postgres `
  -e POSTGRES_DB=hdu_openclaw `
  -p 5432:5432 `
  -v hdu_openclaw_postgres_data:/var/lib/postgresql/data `
  postgres:16
```

### 2. 准备配置

```powershell
Copy-Item .env.example .env
```

编辑 `.env`，至少填好：

- `FEISHU_APP_ID`
- `FEISHU_APP_SECRET`
- `FEISHU_VERIFICATION_TOKEN`
- `LLM_BASE_URL`
- `LLM_API_KEY`
- `LLM_MODEL`
- `DATABASE_URL`

### 3. 本地启动 Go 服务

当前开发阶段推荐本地启动，调试日志最直接：

```powershell
go run ./cmd/server
```

服务启动后会自动：

- 连接 PostgreSQL
- 自动创建 `reminder_tasks` 表
- 启动 reminder scheduler
- 监听 Feishu webhook

### 4. 启动内网穿透

飞书联调时，需要把本地 `8080` 暴露出去，例如使用 `natapp`：

```powershell
natapp -authtoken=你的authtoken
```

然后把飞书事件订阅地址配置成：

```text
http://你的公网地址/webhook/feishu/event
```

## 支持的提醒表达

第一版提醒功能建议先用这些表达联调：

- `五分钟后提醒我喝药`
- `明天早上8点提醒我吃早餐`
- `3.30早上8点提醒我吃早餐`
- `每天晚上10点提醒我背单词`

## 飞书侧配置建议

- 事件订阅回调地址：`http://你的公网域名/webhook/feishu/event`
- 事件订阅类型：`im.message.receive_v1`
- 当前版本建议先不要启用事件加密

## 本地开发建议

- Go 服务：本地启动
- PostgreSQL：Docker 启动
- pgAdmin：作为数据库管理工具
- natapp：作为飞书回调穿透工具

## Docker（仅运行 Go 服务）

```powershell
docker build -t hdu-openclaw:dev .
docker run --rm -p 8080:8080 --env-file .env hdu-openclaw:dev
```
