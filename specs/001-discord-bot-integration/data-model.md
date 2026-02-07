# Data Model: Discord Bot Integration

**Feature Branch**: `001-discord-bot-integration`
**Date**: 2026-02-07

## New Entities

### DiscordConfig

Configuration for Discord bot integration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| bot_token | string | Yes (if using Discord) | Discord Bot authentication token |
| channel_id | string | Yes (if using Discord) | Target Discord channel ID for messages |

**Validation Rules**:
- If `discord` section exists, both `bot_token` and `channel_id` are required
- `bot_token` must be a non-empty string
- `channel_id` must be a non-empty string (Discord channel IDs are numeric strings)

**Relationship**:
- Part of main `Config` struct
- Optional section - entire feature is disabled if missing

## Modified Entities

### Config (config.go)

Add optional Discord configuration to existing Config struct.

| Field | Type | Required | Description | Status |
|-------|------|----------|-------------|--------|
| CoinGecko | struct | Yes | CoinGecko API settings | Existing |
| Email | struct | No* | SMTP email settings | Modified (now optional) |
| Proxy | struct | No | HTTP proxy settings | Existing |
| Coins | []string | Yes | Coin IDs to track | Existing |
| Schedule | struct | Yes | Daily schedule time | Existing |
| **Discord** | struct | No | Discord bot settings | **New** |

*Note: Email becomes optional when Discord is configured (at least one notification channel required)

### Validation Changes

Current validation requires all email fields. New validation logic:

```
IF Discord is configured:
    Validate Discord.bot_token is not empty
    Validate Discord.channel_id is not empty

IF Email is configured:
    Validate Email fields (existing logic)

IF neither Discord nor Email is configured:
    Return error: "at least one notification channel (email or discord) must be configured"
```

## New Interfaces

### Notifier Interface

Abstract notification sending to support multiple channels.

```go
type Notifier interface {
    // Send sends a notification with the given subject and content
    Send(subject string, content interface{}) error

    // IsConfigured returns true if this notifier is properly configured
    IsConfigured() bool

    // Name returns the notifier type name for logging
    Name() string
}
```

**Implementors**:
- `EmailSender` (existing, adapt to interface)
- `DiscordSender` (new)

### DiscordReport

Structured representation of price data for Discord embed.

| Field | Type | Description |
|-------|------|-------------|
| Title | string | Report title with date |
| Color | int | Embed color (hex as int) |
| Fields | []EmbedField | One field per cryptocurrency |
| Footer | string | Data source attribution |
| Timestamp | string | ISO 8601 timestamp |

**EmbedField**:
| Field | Type | Description |
|-------|------|-------------|
| Name | string | Coin name and symbol |
| Value | string | Price, 24h change, market cap |
| Inline | bool | Display inline (true for compact layout) |

## State Transitions

### Notification Flow

```
[Scheduler triggers]
    → [Fetch coin prices]
    → [Generate reports]
        → [HTML report for email]
        → [Embed report for Discord]
    → [Send notifications in parallel]
        → [Email: success/failure (logged)]
        → [Discord: success/failure (logged)]
    → [Log final status]
```

### Configuration Loading Flow

```
[Load YAML]
    → [Parse all sections]
    → [Check notification channels]
        → IF discord.bot_token present → validate Discord config
        → IF email.smtp_server present → validate Email config
        → IF neither present → ERROR
    → [Return Config]
```
