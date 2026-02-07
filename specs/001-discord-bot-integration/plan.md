# Implementation Plan: Discord Bot 集成

**Branch**: `001-discord-bot-integration` | **Date**: 2026-02-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-discord-bot-integration/spec.md`
**Development Mode**: TDD (Test-Driven Development)

## Summary

为 CoinDaily 添加 Discord Bot 集成，使系统在定时任务触发时能够同时发送邮件和 Discord 消息。Discord 为可选功能，仅在配置文件中存在有效的 Bot Token 和频道 ID 时才启用。使用 DiscordGo 库实现 Discord REST API 调用，遵循 TDD 开发模式。

## Technical Context

**Language/Version**: Go 1.21
**Primary Dependencies**:
- `gopkg.in/yaml.v2` (existing)
- `github.com/bwmarrin/discordgo` (new - for Discord API)
**Storage**: N/A (stateless application)
**Testing**: Go standard testing package (`go test`)
**Target Platform**: Linux server (Raspberry Pi)
**Project Type**: Single CLI application
**Performance Goals**: Send notifications within 30 seconds of schedule trigger
**Constraints**: Must work through HTTP proxy, independent channel failure handling
**Scale/Scope**: Single user, 5-10 coins, 2 notification channels

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Note**: Constitution file contains template placeholders. Applying reasonable defaults:

| Principle | Status | Notes |
|-----------|--------|-------|
| TDD (User requested) | ✅ Pass | TDD development mode specified by user |
| Simplicity | ✅ Pass | Minimal new code, reuses existing patterns |
| Backward Compatibility | ✅ Pass | Existing email config remains unchanged |
| Independent Testing | ✅ Pass | Each component is independently testable |

## Project Structure

### Documentation (this feature)

```text
specs/001-discord-bot-integration/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output - Discord library research
├── data-model.md        # Phase 1 output - Config and entity changes
├── quickstart.md        # Phase 1 output - Setup guide
├── contracts/           # Phase 1 output
│   └── discord-api.md   # Discord API contract
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
# Existing structure (single project)
.
├── main.go              # Entry point (minor changes)
├── config.go            # Config struct and loading (add Discord section)
├── config_test.go       # NEW: Config tests
├── coingecko.go         # CoinGecko API client (unchanged)
├── email.go             # Email sender (unchanged)
├── report.go            # Report generator (add Discord embed method)
├── report_test.go       # NEW: Report tests
├── scheduler.go         # Scheduler (add Discord notification)
├── scheduler_test.go    # NEW: Scheduler tests
├── discord.go           # NEW: Discord client
├── discord_test.go      # NEW: Discord tests
├── go.mod               # Add discordgo dependency
├── go.sum               # Updated dependencies
└── config.yaml          # Add optional discord section
```

**Structure Decision**: Flat file structure matching existing codebase pattern. All Go files in root directory, no subdirectories.

## TDD Implementation Order

遵循 TDD 开发模式，按以下顺序实现：

### Phase 1: Config Tests & Implementation
1. **Test First**: 编写 `config_test.go` 测试 Discord 配置解析
   - 测试 Discord 配置存在时正确解析
   - 测试 Discord 配置不存在时应用正常启动
   - 测试仅 Discord 配置（无邮件）时验证通过
   - 测试两者都不配置时报错
2. **Implement**: 修改 `config.go` 添加 Discord 配置结构

### Phase 2: Discord Client Tests & Implementation
1. **Test First**: 编写 `discord_test.go` 测试 Discord 客户端
   - 测试客户端创建
   - 测试消息发送成功场景（使用 mock）
   - 测试认证失败场景
   - 测试权限不足场景
   - 测试重试逻辑
2. **Implement**: 编写 `discord.go` Discord 客户端

### Phase 3: Report Generator Tests & Implementation
1. **Test First**: 编写 `report_test.go` 测试 Discord Embed 生成
   - 测试 Embed 结构正确性
   - 测试多币种格式化
   - 测试涨跌颜色区分
   - 测试字符限制处理
2. **Implement**: 修改 `report.go` 添加 `GenerateDiscordEmbed` 方法

### Phase 4: Scheduler Integration Tests & Implementation
1. **Test First**: 编写 `scheduler_test.go` 测试多渠道通知
   - 测试同时发送邮件和 Discord
   - 测试 Discord 失败不影响邮件
   - 测试邮件失败不影响 Discord
   - 测试仅配置 Discord 时正常工作
2. **Implement**: 修改 `scheduler.go` 集成 Discord 通知

### Phase 5: Integration Testing
1. 端到端测试（手动验证）
2. 使用 `-once` 模式测试完整流程
3. 验证 Discord 频道收到正确格式的消息

## Key Design Decisions

### 1. DiscordGo 库
使用 `github.com/bwmarrin/discordgo` 作为 Discord API 客户端，这是 Go 生态中最成熟的 Discord 库。

### 2. REST-only 模式
仅使用 REST API 发送消息，不需要 WebSocket 网关连接。这简化了实现并减少资源消耗。

### 3. 可选配置
Discord 配置为可选项。如果配置文件中没有 `discord` 部分，系统正常运行但跳过 Discord 通知。

### 4. 独立失败处理
邮件和 Discord 发送独立执行，一个失败不影响另一个。所有失败都记录到日志。

### 5. 代理复用
Discord 客户端复用现有的代理配置，与 CoinGecko 客户端使用相同的代理设置。

## Complexity Tracking

无 Constitution 违规需要记录。
