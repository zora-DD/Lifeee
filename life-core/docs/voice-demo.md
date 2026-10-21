# 语音生活记录 Demo

## 调用流程

1. 小程序录制不超过 60 秒的 MP3。
2. 小程序读取临时文件并转为 Base64。
3. 调用 `POST /v1/activity_log/voice/recognize`。
4. 后端调用腾讯云一句话识别，再调用 MiniMax 整理文本。
5. 前端展示草稿，用户确认后调用已有的 `POST /v1/activity_log/create` 保存。

识别接口本身不会写数据库，避免错误识别直接生成生活记录。

## 服务器配置

在服务器 `conf.yaml` 中增加以下配置。密钥不要提交到 Git：

```yaml
voice:
  enabled: true
  max_audio_bytes: 2097152
  tencent_asr:
    secret_id: "填写腾讯云 SecretId"
    secret_key: "填写腾讯云 SecretKey"
    region: ap-guangzhou
    engine_model_type: 16k_zh
  minimax:
    endpoint: https://api.minimaxi.com/anthropic/v1/messages
    api_key: "填写 MiniMax API Key"
    model: MiniMax-M2.7
    timeout: 45s
```

## 请求

```http
POST /v1/activity_log/voice/recognize
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "audioBase64": "<录音文件的 Base64 内容>",
  "voiceFormat": "mp3"
}
```

支持格式：`wav`、`pcm`、`ogg-opus`、`speex`、`silk`、`mp3`、`m4a`、`aac`、`amr`。

## 响应

```json
{
  "rawText": "今天早上我收拾了客厅，花了半小时。",
  "content": "收拾客厅",
  "occurredAt": "2026-09-03T08:00:00+08:00",
  "durationMinutes": 30,
  "audioDurationMs": 5200,
  "asrRequestId": "腾讯云请求 ID"
}
```
