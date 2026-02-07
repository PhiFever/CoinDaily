# Feature Specification: Discord Bot 集成

**Feature Branch**: `001-discord-bot-integration`
**Created**: 2026-02-07
**Status**: Draft
**Input**: User description: "我希望为这个程序添加discord bot集成，让它在每天定时任务执行时同步发送邮件和discord消息（如果配置文件中有discord bot token)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 配置 Discord 通知 (Priority: P1)

作为 CoinDaily 的用户，我希望在配置文件中添加 Discord Bot 的配置信息，这样系统就能在定时任务触发时向我指定的 Discord 频道发送价格报表。

**Why this priority**: 这是整个功能的核心，没有正确的配置就无法实现 Discord 通知。

**Independent Test**: 可以通过在配置文件中添加 Discord 配置项，然后运行 `-once` 模式验证是否能成功发送 Discord 消息。

**Acceptance Scenarios**:

1. **Given** 配置文件中包含有效的 Discord Bot Token 和频道 ID，**When** 用户启动应用程序，**Then** 系统正确加载 Discord 配置并准备发送消息
2. **Given** 配置文件中未包含 Discord 配置，**When** 用户启动应用程序，**Then** 系统正常启动但不尝试发送 Discord 消息

---

### User Story 2 - 定时发送 Discord 报表 (Priority: P1)

作为 CoinDaily 的用户，我希望在每日定时任务执行时，系统能够同时发送邮件和 Discord 消息，这样我可以在多个渠道接收加密货币价格报表。

**Why this priority**: 这是用户核心需求，与邮件发送并行工作是功能的关键价值点。

**Independent Test**: 可以通过运行 `-once` 模式并同时检查邮箱和 Discord 频道来验证两个渠道都收到了报表。

**Acceptance Scenarios**:

1. **Given** 配置文件中同时配置了邮件和 Discord，**When** 定时任务触发，**Then** 系统同时向邮箱和 Discord 频道发送报表
2. **Given** Discord 发送失败但邮件配置正确，**When** 定时任务触发，**Then** 邮件仍然正常发送，并记录 Discord 发送失败的日志
3. **Given** 邮件发送失败但 Discord 配置正确，**When** 定时任务触发，**Then** Discord 消息仍然正常发送，并记录邮件发送失败的日志

---

### User Story 3 - 仅使用 Discord 通知 (Priority: P2)

作为 CoinDaily 的用户，我希望可以只配置 Discord 而不配置邮件，这样我可以灵活选择只通过 Discord 接收报表。

**Why this priority**: 提供灵活性，让用户可以选择单一通知渠道。

**Independent Test**: 可以通过移除邮件配置只保留 Discord 配置，然后运行 `-once` 模式验证只有 Discord 收到消息。

**Acceptance Scenarios**:

1. **Given** 配置文件中只配置了 Discord（无邮件配置），**When** 定时任务触发，**Then** 系统仅向 Discord 发送报表且不报错
2. **Given** 配置文件中邮件和 Discord 都未配置，**When** 用户启动应用程序，**Then** 系统提示警告但仍可运行

---

### Edge Cases

- 当 Discord Bot Token 无效或过期时，系统应记录错误日志并继续尝试其他通知渠道
- 当 Discord 频道 ID 无效或 Bot 无权限发送消息时，系统应记录详细错误信息
- 当网络连接问题导致 Discord 发送失败时，系统应进行合理的重试
- 当 CoinGecko API 获取数据失败时，系统不应尝试发送空报表到任何渠道
- 当报表内容过长超过 Discord 消息限制时，系统应适当处理（分段或使用 Embed）

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 支持在配置文件中添加 Discord 配置部分，包含 Bot Token 和目标频道 ID
- **FR-002**: 系统 MUST 在 Discord 配置存在且有效时，于定时任务触发时发送 Discord 消息
- **FR-003**: 系统 MUST 支持邮件和 Discord 两个通知渠道独立工作，一个失败不影响另一个
- **FR-004**: 系统 MUST 将加密货币价格报表格式化为适合 Discord 显示的格式（富文本或 Embed）
- **FR-005**: 系统 MUST 在 Discord 配置不存在时，跳过 Discord 发送并正常运行其他功能
- **FR-006**: 系统 MUST 支持 Discord 消息发送使用与 API 请求相同的代理配置
- **FR-007**: 系统 MUST 记录 Discord 消息发送的成功或失败状态到日志
- **FR-008**: 系统 MUST 在 `-once` 模式下同样支持 Discord 消息发送

### Key Entities

- **Discord 配置**: 包含 Bot Token、目标频道 ID，以及可选的代理设置
- **Discord 报表**: 加密货币价格数据的 Discord 友好格式表示，需要包含与邮件报表相同的核心信息
- **通知渠道**: 邮件和 Discord 作为两个独立的通知渠道，可单独或同时使用

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 用户在添加 Discord 配置后，首次运行时能在 30 秒内收到 Discord 消息
- **SC-002**: 在两个通知渠道都配置的情况下，定时任务能够在 1 分钟内完成所有通知发送
- **SC-003**: Discord 消息内容与邮件报表包含相同的核心价格数据（币种、价格、24h变化）
- **SC-004**: 单个通知渠道失败时，其他渠道成功率保持 100%
- **SC-005**: 用户无需修改现有邮件配置即可添加 Discord 功能（向后兼容）

## Assumptions

- Discord Bot 已由用户预先创建并获取到有效的 Bot Token
- Discord Bot 已被添加到目标服务器并拥有目标频道的发送消息权限
- 用户了解如何获取 Discord 频道 ID
- 系统运行环境能够访问 Discord API（直接或通过代理）
