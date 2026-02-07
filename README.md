# CoinDaily - 每日加密货币价格报表

CoinDaily 是一个用 Go 语言编写的工具，它会每天自动获取加密货币价格数据并生成精美的报表，发送到您的邮箱和/或 Discord 频道。

## 功能特性

- 🚀 自动获取 CoinGecko API 的加密货币价格数据
- 📊 生成美观的 HTML 格式报表（邮件）和 Embed 格式报表（Discord）
- 📧 支持邮件自动发送
- 🤖 支持 Discord Bot 消息推送
- ⏰ 支持定时任务调度
- 🔧 灵活的 YAML 配置文件
- 💰 支持多种加密货币追踪
- 🔀 支持多通知渠道（邮件和 Discord 可独立配置）

## 快速开始

### 1. 配置邮箱设置

参见 `config.yaml.example` 编辑 `config.yaml` 文件：

### 2. 编译并运行

```bash
# 编译项目
go build -o coindaily

# 运行（持续运行，按计划发送）
./coindaily

# 单次运行（立即发送一次报表）
./coindaily -once

# 指定配置文件
./coindaily -config /path/to/config.yaml
```

## 命令行选项

- `-config`: 指定配置文件路径（默认：config.yaml）
- `-once`: 单次运行模式，生成报表后退出（默认：false）

## 支持的加密货币

您可以在配置文件中添加任何 CoinGecko 支持的加密货币 ID，常见的包括：

- bitcoin
- ethereum
- binancecoin
- cardano
- solana
- polkadot
- dogecoin
- avalanche-2
- polygon-token
- chainlink

完整列表请参考 [CoinGecko API 文档](https://docs.coingecko.com/v3.0.1/reference/endpoint-overview)

## Gmail 配置说明

如果使用 Gmail，需要：

1. 启用两步验证
2. 生成[应用密码](https://support.google.com/mail/answer/185833?hl=en#zippy=%2Cwhy-you-may-need-an-app-password)（不是您的常规密码）
3. 在配置文件中使用应用密码

## Discord 配置说明

要使用 Discord 通知功能，需要：

1. 在 [Discord Developer Portal](https://discord.com/developers/applications) 创建应用
2. 在应用中创建 Bot，获取 Bot Token
3. 将 Bot 添加到您的服务器（需要 "Send Messages" 和 "Embed Links" 权限）
4. 获取目标频道的 ID（在 Discord 中启用开发者模式，右键频道复制 ID）
5. 在配置文件中添加：

```yaml
discord:
  bot_token: "your_bot_token"
  channel_id: "your_channel_id"
```

**注意**：至少需要配置邮件或 Discord 其中一个通知渠道。两者可以同时配置，也可以只配置其中一个。

## 报表内容

每日报表包含以下信息：

- 币种名称和符号
- 当前价格（USD）
- 24小时价格变化
- 24小时价格变化率
- 市值
- 24小时交易量