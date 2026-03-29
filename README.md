# hdu-openclaw

杭电龙虾的 Go 版最小可用后端。当前版本只实现一条最小链路：

- 飞书机器人接收文本消息
- 调用 OpenAI-compatible 大模型 API
- 将回复发回飞书

## 本地运行

1. 参考 `.env.example` 准备配置
2. 在当前终端设置飞书与大模型环境变量
3. 启动服务：

```powershell
$env:APP_ADDR=':8080'
$env:FEISHU_APP_ID='cli_xxx'
$env:FEISHU_APP_SECRET='replace_me'
$env:FEISHU_VERIFICATION_TOKEN='replace_me'
$env:LLM_BASE_URL='https://api.openai.com/v1'
$env:LLM_API_KEY='replace_me'
$env:LLM_MODEL='gpt-4o-mini'
go run ./cmd/server
```

## 飞书侧配置建议

- 事件订阅回调地址：`https://你的公网域名/webhook/feishu/event`
- 事件订阅类型：`im.message.receive_v1`
- v0.1 建议先不要启用消息加密

## Docker

```powershell
Copy-Item .env.example .env
# 编辑 .env，填入真实配置
docker build -t hdu-openclaw:dev .
docker run --rm -p 8080:8080 --env-file .env hdu-openclaw:dev
```
