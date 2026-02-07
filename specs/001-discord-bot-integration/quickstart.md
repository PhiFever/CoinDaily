# Quickstart: Discord Bot Integration

**Feature Branch**: `001-discord-bot-integration`
**Date**: 2026-02-07

## Prerequisites

1. Go 1.21 or later installed
2. Existing CoinDaily application working
3. Discord account with server admin access

## Discord Bot Setup

### Step 1: Create Discord Application

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application"
3. Name it "CoinDaily Bot" (or preferred name)
4. Click "Create"

### Step 2: Create Bot

1. In your application, go to "Bot" section
2. Click "Add Bot"
3. Under "Token", click "Copy" to get your bot token
4. Save this token securely - you'll need it for configuration

### Step 3: Configure Bot Permissions

1. In "Bot" section, scroll to "Privileged Gateway Intents"
2. No special intents needed for message sending
3. Go to "OAuth2" → "URL Generator"
4. Select scopes: `bot`
5. Select permissions: `Send Messages`, `Embed Links`
6. Copy the generated URL

### Step 4: Add Bot to Server

1. Open the generated OAuth2 URL in browser
2. Select your Discord server
3. Authorize the bot

### Step 5: Get Channel ID

1. In Discord, enable Developer Mode (Settings → Advanced → Developer Mode)
2. Right-click the channel where you want reports
3. Click "Copy ID"

## Configuration

Add the following to your `config.yaml`:

```yaml
# Existing configuration...

# Discord 配置 (可选)
discord:
  bot_token: "your-bot-token-here"
  channel_id: "your-channel-id-here"
```

## Testing

### Test Single Report

```bash
# Build the application
go build -o coindaily

# Run once to test
./coindaily -once
```

Expected output:
```
CoinDaily - 每日加密货币价格报表工具启动中...
配置加载成功，将跟踪 5 个加密货币
每日报表发送时间: 09:00
报表将发送至: [email@example.com]
Discord 通知已启用，频道 ID: 123456789
单次运行模式，生成并发送报表后退出...
开始生成每日加密货币价格报表...
成功获取到 5 个加密货币的价格数据
每日报表已成功发送到邮箱
Discord 消息已成功发送
```

### Verify in Discord

Check your Discord channel for an embed message with:
- Title: 🚀 每日加密货币价格报表
- Fields showing each cryptocurrency with price and 24h change
- Footer showing data source

## Troubleshooting

### "Discord 发送失败: 401 Unauthorized"
- Bot token is invalid or expired
- Regenerate token in Discord Developer Portal

### "Discord 发送失败: 403 Forbidden"
- Bot lacks permission to send messages
- Re-invite bot with correct permissions
- Check channel permissions

### "Discord 发送失败: 404 Not Found"
- Channel ID is invalid
- Ensure Developer Mode is enabled when copying ID

### No Discord message sent (no error)
- Check if `discord` section exists in config
- Verify both `bot_token` and `channel_id` are present

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestDiscord
```

## Dependencies

New dependency added:
```bash
go get github.com/bwmarrin/discordgo
```

This is automatically handled when building:
```bash
go mod tidy
go build -o coindaily
```
