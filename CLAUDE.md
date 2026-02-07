# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 构建和运行命令

```bash
# 构建项目
go build -o coindaily

# 定时运行模式（启动时立即发送一次，之后按配置时间发送）
./coindaily

# 单次运行模式（发送一次后退出，适合测试）
./coindaily -once

# 指定配置文件
./coindaily -config /path/to/config.yaml
```

## 架构说明

CoinDaily 是一个 Go 应用程序，从 CoinGecko API 获取加密货币价格，并发送每日报表到邮件和/或 Discord。

### 核心组件

- **main.go** - 入口，处理命令行参数 (-config, -once) 和信号处理（优雅关闭）
- **config.go** - YAML 配置加载和验证（Config 结构体）
- **coingecko.go** - CoinGecko API 客户端，支持可选代理（CoinGeckoClient）
- **report.go** - 报表生成（HTML 和 Discord Embed）（ReportGenerator）
- **email.go** - SMTP 邮件发送（EmailSender）
- **discord.go** - Discord Bot 消息发送（DiscordSender）
- **scheduler.go** - 定时调度器，支持多渠道通知（Scheduler）

### 数据流程

1. Scheduler 在配置的 hour:minute 触发（每分钟检查一次）
2. CoinGeckoClient 通过 `/coins/markets` 端点获取配置的币种价格
3. ReportGenerator 生成报表：
   - HTML 格式用于邮件
   - Discord Embed 格式用于 Discord
4. 通知发送（独立执行，互不影响）：
   - EmailSender 通过 SMTP 发送 HTML 格式邮件（如果配置了邮件）
   - DiscordSender 通过 Discord API 发送 Embed 消息（如果配置了 Discord）

### 配置说明

使用 `config.yaml` 配置文件，包含以下部分：
- `coingecko.api_key` - 必需，CoinGecko demo API 密钥
- `email.*` - SMTP 服务器设置和收件人（可选，如果配置了 Discord）
- `discord.bot_token` - Discord Bot Token（可选，如果配置了邮件）
- `discord.channel_id` - Discord 频道 ID（可选，如果配置了邮件）
- `proxy.*` - 可选的 HTTP 代理（用于 API 和 Discord 请求）
- `coins` - 要追踪的 CoinGecko 币种 ID 列表
- `schedule.hour/minute` - 每日报表发送时间（24小时制）

**注意**：至少需要配置一个通知渠道（邮件或 Discord）

### API 细节

- 使用 CoinGecko demo API，请求头为 `x-cg-demo-api-key`
- 通过 `/api/v3/coins/markets` 获取 USD 计价的价格数据

## Active Technologies
- Go 1.21 (001-discord-bot-integration)
- N/A (stateless application) (001-discord-bot-integration)

## Recent Changes
- 001-discord-bot-integration: Added Go 1.21
