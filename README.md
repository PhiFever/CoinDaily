# CoinDaily - 多市场行情报表

CoinDaily 是一个 Go 编写的行情通知工具。每次运行会把已配置的 CoinGecko 加密货币、Alpaca 美股/ETF，以及 Hyperliquid 永续合约合并成一份报表，发送到邮箱和/或 Discord。

## 功能特性

- 🚀 自动获取 CoinGecko API 的加密货币价格数据
- 🇺🇸 获取 Alpaca 15 分钟延迟 SIP 美股/ETF 行情
- ♾️ 获取 Hyperliquid 公开永续行情（包括 HIP-3 合约）
- 💸 展示成交额、市值变化、名义持仓变化与资金费率等“资金动向”代理指标
- 📊 生成美观的 HTML 格式报表（邮件）和 Embed 格式报表（Discord）
- 📧 支持邮件自动发送
- 🤖 支持 Discord Bot 消息推送
- ⏰ 支持内置定时任务和外部 cron 单次运行
- 🔧 灵活的 YAML 配置文件
- 💰 支持多种加密货币追踪
- 🔀 支持多通知渠道（邮件和 Discord 可独立配置）

## 快速开始

### 1. 创建配置

复制 `config.yaml.example` 为 `config.yaml` 并填写需要启用的服务。至少配置一种市场和一种通知渠道。

新增市场的最小配置如下：

```yaml
alpaca:
  api_key: "your_alpaca_api_key_here"
  secret_key: "your_alpaca_secret_key_here"
  feed: "delayed_sip"

stocks:
  - "QQQ"
  - "SPCX"

hyperliquid:
  perpetuals:
    - "xyz:ZHIPU"
```

`stocks` 为空时不需要 Alpaca 密钥。Hyperliquid Info API 是公开接口，不需要钱包、私钥或账户信息。旧的纯 CoinGecko 配置仍然有效。

### 2. 编译并运行

```bash
# 编译项目
go build -o coindaily

# 运行（持续运行，按 schedule 发送）
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

### 如何查找正确的 API ID

注意：CoinGecko 网站的 URL 和 API ID 不一定相同（例如 BNB 的网址是 `/coins/bnb`，但 API ID 是 `binancecoin`）。查找正确 ID 有两种方式：

**方法一：查询完整币种列表**

在浏览器访问以下地址（无需 API key），搜索币种名称或 symbol 即可找到对应的 `id`：

```
https://api.coingecko.com/api/v3/coins/list
```

**方法二：查看币种页面底部**

在 CoinGecko 每个币种页面（从[此处](https://www.coingecko.com/en/all-cryptocurrencies)点击进入）滚动到底部，找到 "API ID" 字段，该值即为配置文件中应填写的 ID。

完整列表请参考 [CoinGecko API 文档](https://docs.coingecko.com/v3.0.1/reference/endpoint-overview)

## 美股 / ETF 行情

股票代码配置在顶层 `stocks` 列表。当前实现固定采用 `alpaca.feed: delayed_sip`：程序通过 Alpaca 历史 REST API 请求截止到至少 15 分钟前的 SIP 综合数据，因此报告不是实时、也不是可成交报价。

股票区块显示：

- 最近有效的延迟 SIP 价格及其时间戳
- 相对上一常规交易日收盘价的变化
- 最近完整小时的 VWAP、成交量、方向和估算成交额

休市时会继续显示最近的有效价格；时间戳用于辨别数据的新鲜度。只有配置了 `stocks` 时才要求 `alpaca.api_key` 和 `alpaca.secret_key`。

## Hyperliquid 永续行情

永续合约配置在 `hyperliquid.perpetuals`，HIP-3 合约必须包含 DEX 前缀，例如 `xyz:ZHIPU`。程序使用无需鉴权的 Hyperliquid Info API，并按 universe 名称动态匹配合约，不依赖固定资产编号。

永续区块显示：

- 标记价格、预言机价格及溢折价
- 相对 `prevDayPx` 的 24 小时变化和当前每小时资金费率
- 名义持仓量，以及较上一份不超过 90 分钟的有效快照的增仓/减仓
- 最近完整小时估算成交额和 24 小时成交额；小时估算使用 candle 基础资产成交量乘以 OHLC 平均代表价

持仓快照保存在活动配置文件旁的 `.coindaily-state.json`，采用临时文件加原子替换写入。文件缺失、损坏或快照过期时，持仓变化显示 `N/A`，不会阻止报表发送。

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

## 报表与故障处理

邮件和 Discord 会收到同一份分区报表。各行情源独立采集：单个来源失败时，健康区块仍会发送，并在失败区块显示警告；只有所有已配置来源都没有可用数据时才不发送通知。邮件与 Discord 也彼此独立尝试发送。

“资金动向”是基于成交额、市值变化、名义持仓和资金费率的观察指标，不等同于真实净资金流，也不会被展示为投资或交易建议。

如果系统使用外部 cron 每小时执行，可保持现有命令不变：

```cron
0 * * * * /path/to/coindaily -once -config /path/to/config.yaml
```
