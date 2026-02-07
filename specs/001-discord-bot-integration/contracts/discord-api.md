# Discord API Contract

**Feature Branch**: `001-discord-bot-integration`
**Date**: 2026-02-07

## Overview

This document defines the contract for Discord message sending using the Discord REST API.

## External API: Discord REST API

### Endpoint: Create Message

**URL**: `POST https://discord.com/api/v10/channels/{channel_id}/messages`

**Headers**:
```
Authorization: Bot {bot_token}
Content-Type: application/json
```

**Request Body** (Embed Message):
```json
{
  "embeds": [
    {
      "title": "🚀 每日加密货币价格报表",
      "description": "2026年02月07日",
      "color": 16766720,
      "fields": [
        {
          "name": "Bitcoin (BTC)",
          "value": "**$45,000.00**\n24h: +2.5% | 市值: $850.2B",
          "inline": true
        },
        {
          "name": "Ethereum (ETH)",
          "value": "**$2,800.00**\n24h: -1.2% | 市值: $320.5B",
          "inline": true
        }
      ],
      "footer": {
        "text": "数据来源: CoinGecko API | CoinDaily 自动生成"
      },
      "timestamp": "2026-02-07T09:00:00Z"
    }
  ]
}
```

**Success Response**: `200 OK`
```json
{
  "id": "message_id",
  "channel_id": "channel_id",
  "content": "",
  "embeds": [...]
}
```

**Error Responses**:
- `401 Unauthorized`: Invalid bot token
- `403 Forbidden`: Bot lacks permission to send messages in channel
- `404 Not Found`: Channel does not exist
- `429 Too Many Requests`: Rate limited (includes retry_after header)

## Internal Contract: DiscordSender

### Interface Definition

```go
// DiscordSender sends cryptocurrency price reports to Discord
type DiscordSender interface {
    // SendReport sends a price report to the configured Discord channel
    // Returns nil on success, error on failure
    SendReport(coins []CoinPrice) error

    // IsConfigured returns true if Discord is properly configured
    IsConfigured() bool
}
```

### Configuration Contract

```yaml
# config.yaml - Discord section
discord:
  bot_token: "Bot authentication token (required if section present)"
  channel_id: "Target channel ID (required if section present)"
```

### Error Contract

| Error Type | Condition | Retry |
|------------|-----------|-------|
| InvalidConfig | Missing token or channel ID | No |
| AuthenticationError | Invalid or expired bot token | No |
| PermissionError | Bot cannot send to channel | No |
| NotFoundError | Channel does not exist | No |
| RateLimitError | Too many requests | Yes (after retry_after) |
| NetworkError | Connection failed | Yes (3 attempts, 10s delay) |

## Internal Contract: ReportGenerator

### New Method

```go
// GenerateDiscordEmbed creates a Discord embed from coin price data
func (r *ReportGenerator) GenerateDiscordEmbed(coins []CoinPrice) *DiscordEmbed
```

### DiscordEmbed Structure

```go
type DiscordEmbed struct {
    Title       string       `json:"title"`
    Description string       `json:"description,omitempty"`
    Color       int          `json:"color"`
    Fields      []EmbedField `json:"fields"`
    Footer      *EmbedFooter `json:"footer,omitempty"`
    Timestamp   string       `json:"timestamp,omitempty"`
}

type EmbedField struct {
    Name   string `json:"name"`
    Value  string `json:"value"`
    Inline bool   `json:"inline"`
}

type EmbedFooter struct {
    Text string `json:"text"`
}
```

## Validation Rules

### Bot Token
- Must be non-empty string
- Format: No specific format validation (opaque to application)
- Validated by Discord API on first use

### Channel ID
- Must be non-empty string
- Discord channel IDs are 17-19 digit numeric strings
- Validated by Discord API on first use

### Embed Limits
- Title: max 256 characters
- Description: max 4096 characters
- Fields: max 25 fields
- Field name: max 256 characters
- Field value: max 1024 characters
- Footer text: max 2048 characters
- Total embed: max 6000 characters
