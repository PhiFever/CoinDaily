# Tasks: Discord Bot 集成

**Input**: Design documents from `/specs/001-discord-bot-integration/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/
**Development Mode**: TDD (测试驱动开发)
**Language**: Go 1.21
**New Dependency**: `github.com/bwmarrin/discordgo`

**Organization**: 任务按用户故事分组，每个故事可以独立实现和测试。遵循 TDD 模式：先写测试 → 测试失败 → 实现功能 → 测试通过。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 任务所属的用户故事（US1, US2, US3）
- 路径基于项目根目录的扁平结构

---

## Phase 1: 项目设置

**Purpose**: 添加依赖和基础测试结构

- [x] T001 添加 discordgo 依赖到 go.mod
- [x] T002 [P] 创建配置测试基础设施 config_test.go（测试辅助函数和临时配置文件创建）

**Checkpoint**: 依赖和测试基础设施准备就绪

---

## Phase 2: 基础设施（所有用户故事的前置条件）

**Purpose**: Config 结构体修改和验证逻辑更新，这是所有 Discord 功能的基础

**⚠️ 关键**: 在此阶段完成前，不能开始任何用户故事的实现

### 测试先行 - Config 模块

- [x] T003 [P] 编写测试：Discord 配置存在时正确解析 config_test.go
- [x] T004 [P] 编写测试：Discord 配置不存在时应用正常启动 config_test.go
- [x] T005 [P] 编写测试：仅 Discord 配置（无邮件）时验证通过 config_test.go
- [x] T006 [P] 编写测试：邮件和 Discord 都未配置时报错 config_test.go

### 实现 - Config 模块

- [x] T007 在 Config 结构体中添加 Discord 配置部分 config.go
- [x] T008 修改 validateConfig 函数支持可选邮件配置 config.go
- [x] T009 运行测试验证 Config 修改正确 (`go test -v -run TestConfig`)

**Checkpoint**: 配置解析和验证完成，所有测试通过

---

## Phase 3: User Story 1 - 配置 Discord 通知 (Priority: P1) 🎯 MVP

**Goal**: 用户可以在配置文件中添加 Discord Bot 配置，系统能正确加载并准备发送消息

**Independent Test**: 运行 `./coindaily -once`，检查日志显示 "Discord 通知已启用"

### 测试先行 - Discord 客户端

> **注意**: 先写这些测试，确保它们 FAIL 后再实现

- [x] T010 [P] [US1] 编写测试：DiscordSender 客户端创建（带代理配置）discord_test.go
- [x] T011 [P] [US1] 编写测试：IsConfigured 方法正确判断配置状态 discord_test.go
- [x] T012 [P] [US1] 编写测试：SendEmbed 成功发送消息（使用 mock HTTP client）discord_test.go

### 测试先行 - Report Generator

- [x] T013 [P] [US1] 编写测试：GenerateDiscordEmbed 生成正确的 Embed 结构 report_test.go
- [x] T014 [P] [US1] 编写测试：Embed 字段包含币种名称、价格、24h变化 report_test.go
- [x] T015 [P] [US1] 编写测试：涨跌使用不同颜色 report_test.go

### 实现 - Discord 客户端

- [x] T016 [US1] 定义 DiscordEmbed、EmbedField、EmbedFooter 结构体 discord.go
- [x] T017 [US1] 实现 DiscordSender 结构体和 NewDiscordSender 构造函数 discord.go
- [x] T018 [US1] 实现 IsConfigured 方法检查 Token 和 ChannelID discord.go
- [x] T019 [US1] 实现 SendEmbed 方法发送 Discord 消息（支持代理）discord.go
- [x] T020 [US1] 实现重试逻辑（3次重试，10秒间隔）discord.go

### 实现 - Report Generator

- [x] T021 [US1] 实现 GenerateDiscordEmbed 方法生成 Discord Embed report.go

### 验证

- [x] T022 [US1] 运行所有 Discord 相关测试 (`go test -v -run "Discord|Embed"`)

**Checkpoint**: User Story 1 完成，Discord 客户端和报表生成器可独立测试

---

## Phase 4: User Story 2 - 定时发送 Discord 报表 (Priority: P1)

**Goal**: 定时任务触发时，系统同时发送邮件和 Discord 消息，两个渠道独立工作

**Independent Test**: 运行 `./coindaily -once`，检查邮箱和 Discord 频道都收到报表

### 测试先行 - Scheduler 集成

- [x] T023 [P] [US2] 编写测试：Scheduler 同时初始化 EmailSender 和 DiscordSender scheduler_test.go
- [x] T024 [P] [US2] 编写测试：runDailyReport 调用两个发送器 scheduler_test.go
- [x] T025 [P] [US2] 编写测试：Discord 发送失败不影响邮件发送 scheduler_test.go
- [x] T026 [P] [US2] 编写测试：邮件发送失败不影响 Discord 发送 scheduler_test.go

### 实现 - Scheduler 集成

- [x] T027 [US2] 在 Scheduler 结构体中添加 discordSender 字段 scheduler.go
- [x] T028 [US2] 修改 NewScheduler 初始化 DiscordSender（仅当配置存在时）scheduler.go
- [x] T029 [US2] 修改 runDailyReport 同时发送邮件和 Discord scheduler.go
- [x] T030 [US2] 实现独立错误处理：一个失败不阻止另一个 scheduler.go
- [x] T031 [US2] 添加 Discord 发送成功/失败的日志记录 scheduler.go

### 实现 - Main 入口点

- [x] T032 [US2] 修改 main.go 显示 Discord 配置状态日志 main.go

### 验证

- [x] T033 [US2] 运行所有 Scheduler 测试 (`go test -v -run TestScheduler`)

**Checkpoint**: User Story 2 完成，邮件和 Discord 可独立工作且互不影响

---

## Phase 5: User Story 3 - 仅使用 Discord 通知 (Priority: P2)

**Goal**: 用户可以只配置 Discord 而不配置邮件，系统正常运行

**Independent Test**: 移除邮件配置，只保留 Discord 配置，运行 `-once` 验证只有 Discord 收到消息

### 测试先行

- [x] T034 [P] [US3] 编写测试：仅 Discord 配置时 Scheduler 正常初始化 scheduler_test.go
- [x] T035 [P] [US3] 编写测试：仅 Discord 配置时 runDailyReport 只发送 Discord scheduler_test.go
- [x] T036 [P] [US3] 编写测试：无任何通知渠道时记录警告但不崩溃 scheduler_test.go

### 实现

- [x] T037 [US3] 修改 Scheduler 跳过未配置的 EmailSender scheduler.go
- [x] T038 [US3] 添加无通知渠道时的警告日志 scheduler.go

### 验证

- [x] T039 [US3] 运行完整测试套件 (`go test -v ./...`)

**Checkpoint**: 所有用户故事完成，系统支持灵活的通知渠道配置

---

## Phase 6: 收尾和边缘情况处理

**Purpose**: 处理边缘情况、完善错误处理、更新文档

### 边缘情况测试

- [x] T040 [P] 编写测试：Discord Bot Token 无效时记录详细错误 discord_test.go
- [x] T041 [P] 编写测试：频道 ID 无效时记录详细错误 discord_test.go
- [x] T042 [P] 编写测试：网络超时时正确重试 discord_test.go

### 边缘情况实现

- [x] T043 实现 Discord API 错误的详细错误消息（401/403/404）discord.go
- [x] T044 处理 Discord 消息长度限制（超过 6000 字符时截断）discord.go

### 文档和配置示例

- [x] T045 [P] 在 config.yaml 中添加 Discord 配置示例（注释形式）config.yaml
- [x] T046 [P] 更新 CLAUDE.md 添加 Discord 配置说明 CLAUDE.md

### 最终验证

- [x] T047 运行完整测试套件并确保 100% 通过 (`go test -v ./...`)
- [x] T048 使用 `-once` 模式进行端到端测试
- [x] T049 构建并验证二进制文件 (`go build -o coindaily`)

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1 (设置)
    ↓
Phase 2 (基础设施) ← 阻塞所有用户故事
    ↓
┌───────────────┬───────────────┐
↓               ↓               ↓
Phase 3 (US1)   Phase 4 (US2)   Phase 5 (US3)
[MVP]           [需要 US1]       [可并行]
└───────────────┴───────────────┘
    ↓
Phase 6 (收尾)
```

### User Story Dependencies

- **User Story 1 (P1)**: Phase 2 完成后可开始 - 不依赖其他用户故事
- **User Story 2 (P1)**: 依赖 US1 的 Discord 客户端和报表生成器
- **User Story 3 (P2)**: 依赖 US2 的 Scheduler 修改

### TDD 执行顺序（每个阶段内）

1. 编写测试 → 验证测试失败 (Red)
2. 实现最小代码使测试通过 (Green)
3. 重构优化 (Refactor)
4. 提交代码

### Parallel Opportunities

**Phase 2 内可并行的测试任务**:
```bash
# 可同时执行：
T003, T004, T005, T006  # Config 相关测试
```

**Phase 3 内可并行的测试任务**:
```bash
# 可同时执行：
T010, T011, T012  # Discord 客户端测试
T013, T014, T015  # Report Generator 测试
```

**Phase 4 内可并行的测试任务**:
```bash
# 可同时执行：
T023, T024, T025, T026  # Scheduler 集成测试
```

---

## Parallel Example: User Story 1

```bash
# 并行启动 Discord 客户端相关测试：
Task: "编写测试：DiscordSender 客户端创建（带代理配置）discord_test.go"
Task: "编写测试：IsConfigured 方法正确判断配置状态 discord_test.go"
Task: "编写测试：SendEmbed 成功发送消息（使用 mock HTTP client）discord_test.go"

# 并行启动 Report Generator 相关测试：
Task: "编写测试：GenerateDiscordEmbed 生成正确的 Embed 结构 report_test.go"
Task: "编写测试：Embed 字段包含币种名称、价格、24h变化 report_test.go"
Task: "编写测试：涨跌使用不同颜色 report_test.go"
```

---

## Implementation Strategy

### MVP First (仅 User Story 1)

1. 完成 Phase 1: 设置
2. 完成 Phase 2: 基础设施 (关键 - 阻塞所有故事)
3. 完成 Phase 3: User Story 1
4. **停止并验证**: 独立测试 Discord 客户端功能
5. 如果满足基本需求可部署

### Incremental Delivery

1. 完成设置 + 基础设施 → 配置解析就绪
2. 添加 User Story 1 → 测试 Discord 客户端 → 可发送消息
3. 添加 User Story 2 → 测试 Scheduler → 邮件+Discord 同步
4. 添加 User Story 3 → 测试灵活配置 → 完整功能

### 推荐执行顺序

由于这是单人项目，建议按顺序执行：
1. Phase 1 → Phase 2（必须）
2. Phase 3（MVP，Discord 客户端）
3. Phase 4（Scheduler 集成）
4. Phase 5（灵活配置）
5. Phase 6（收尾）

---

## Notes

- 所有代码注释使用中文
- 日志消息使用中文
- 错误消息使用中文（与现有代码风格一致）
- [P] 任务 = 不同文件，无依赖
- [Story] 标签用于追踪任务属于哪个用户故事
- TDD 模式：每个测试任务后都应验证测试失败再进行实现
- 每个任务或逻辑组完成后提交代码
- 任何 Checkpoint 都可以停止验证故事独立性
