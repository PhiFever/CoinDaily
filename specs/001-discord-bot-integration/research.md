# Research: Discord Bot Integration

**Feature Branch**: `001-discord-bot-integration`
**Date**: 2026-02-07

## Discord Go Library Selection

### Decision: DiscordGo (`github.com/bwmarrin/discordgo`)

### Rationale
- Most popular and widely adopted Discord library for Go
- Well-documented with extensive examples
- Active community support (Discord Gophers chat server)
- Mature and stable API
- Supports all Discord API features including message sending, embeds, and webhooks
- Simple API for sending messages to channels

### Alternatives Considered

| Library | Pros | Cons | Reason Not Chosen |
|---------|------|------|-------------------|
| [Arikawa](https://github.com/diamondburned/arikawa) | Modular design, typed snowflakes, pluggable cache | More complex, less community resources | Over-engineered for simple message sending use case |
| [Goscord](https://goscord.dev/) | Simple API | Smaller community, less documentation | Less proven in production |
| Direct Discord REST API | No dependency | More code to maintain, handle auth/retry | Reinventing the wheel |

### Implementation Approach

For this feature, we only need to send messages to a channel - no need for gateway/websocket connections. DiscordGo supports both:
1. **Bot with Gateway**: Full bot with real-time events (unnecessary for our use case)
2. **REST-only**: Direct HTTP calls to send messages (sufficient for our use case)

**Chosen**: REST-only approach using DiscordGo's `ChannelMessageSendEmbed` for rich formatted messages.

## Discord Message Format

### Decision: Discord Embed

### Rationale
- Embeds provide rich formatting similar to HTML email
- Support for colors, fields, titles, footers
- Better visual presentation than plain text
- Can include multiple fields for each cryptocurrency
- 2000 character limit for regular messages vs 6000 for embed descriptions

### Embed Structure for Price Report
```
Title: 🚀 每日加密货币价格报表
Color: Theme color (e.g., gold/green)
Fields: One field per coin with price info
Footer: Data source and timestamp
```

## Proxy Support

### Decision: Reuse existing proxy configuration pattern

### Rationale
- Existing `CoinGeckoClient` already implements proxy support
- Same pattern can be applied to Discord client
- Consistent user experience - single proxy config for all outbound connections

## Error Handling

### Decision: Independent channel failure handling

### Rationale
- Email and Discord should fail independently
- Log errors but don't block other notifications
- Retry logic similar to CoinGecko client (3 attempts with 10s delay)

## TDD Approach

### Decision: Test-Driven Development with interfaces

### Rationale
- User requested TDD development mode
- Create interfaces for Discord client to enable mocking
- Write tests first for:
  1. Config parsing with Discord fields
  2. Discord client initialization
  3. Report formatting for Discord
  4. Scheduler integration with multiple notifiers
  5. Error handling scenarios

### Test Structure
```
discord_test.go      # Unit tests for Discord client
config_test.go       # Tests for Discord config parsing
scheduler_test.go    # Integration tests for multi-channel notification
report_test.go       # Tests for Discord report formatting
```

## Configuration Structure

### Decision: Optional `discord` section in config.yaml

```yaml
discord:
  bot_token: "your-bot-token"
  channel_id: "channel-id"
```

### Rationale
- Follows existing config pattern (coingecko, email, proxy sections)
- Optional - if missing, Discord notification is skipped
- Uses same proxy settings as other HTTP clients

## Sources

- [DiscordGo GitHub](https://github.com/bwmarrin/discordgo)
- [DiscordGo Package Documentation](https://pkg.go.dev/github.com/bwmarrin/discordgo)
- [Arikawa GitHub](https://github.com/diamondburned/arikawa)
- [Goscord](https://goscord.dev/)
