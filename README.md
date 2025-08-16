# CoinDaily - 每日加密货币价格报表

CoinDaily 是一个用 Go 语言编写的工具，它会每天自动获取加密货币价格数据并生成精美的 HTML 报表发送到您的邮箱。

## 功能特性

- 🚀 自动获取 CoinGecko API 的加密货币价格数据
- 📊 生成美观的 HTML 格式报表
- 📧 支持邮件自动发送
- ⏰ 支持定时任务调度
- 🔧 灵活的 YAML 配置文件
- 💰 支持多种加密货币追踪

## 快速开始

### 1. 配置邮箱设置

编辑 `config.yaml` 文件：

```yaml
email:
  smtp_server: "smtp.gmail.com"
  smtp_port: 587
  username: "your-email@gmail.com"
  password: "your-app-password"  # 对于Gmail，需要使用应用密码
  to: "recipient@example.com"

coins:
  - "bitcoin"
  - "ethereum" 
  - "binancecoin"
  - "cardano"
  - "solana"

schedule:
  hour: 9    # 每天9点
  minute: 0  # 0分发送
```

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
2. 生成应用密码（不是您的常规密码）
3. 在配置文件中使用应用密码

## 报表内容

每日报表包含以下信息：

- 币种名称和符号
- 当前价格（USD）
- 24小时价格变化
- 24小时价格变化率
- 市值
- 24小时交易量

## 项目结构

```
CoinDaily/
├── main.go          # 主程序入口
├── config.go        # 配置文件处理
├── coingecko.go     # CoinGecko API 客户端
├── email.go         # 邮件发送功能
├── report.go        # HTML 报表生成
├── scheduler.go     # 定时任务调度
├── config.yaml      # 配置文件
├── go.mod           # Go 模块文件
└── README.md        # 说明文档
```

## 许可证

本项目基于 MIT 许可证开源。