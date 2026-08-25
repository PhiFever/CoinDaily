package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	once := flag.Bool("once", false, "只运行一次，不启动定时任务")
	flag.Parse()

	log.Println("CoinDaily - 市场行情报表工具启动中...")

	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	log.Printf("配置加载成功：加密货币 %d，美股/ETF %d，永续合约 %d", len(config.Coins), len(config.Stocks), len(config.Hyperliquid.Perpetuals))
	log.Printf("报表发送时间: %02d:%02d", config.Schedule.Hour, config.Schedule.Minute)

	// 显示通知渠道状态
	if isEmailConfigured(config) {
		log.Printf("邮件通知已启用，收件人: %v", config.Email.To)
	} else {
		log.Println("邮件通知未配置")
	}

	if isDiscordConfigured(config) {
		log.Printf("Discord 通知已启用，频道 ID: %s", config.Discord.ChannelID)
	} else {
		log.Println("Discord 通知未配置")
	}

	scheduler := NewScheduler(config)

	if *once {
		log.Println("单次运行模式，生成并发送报表后退出...")
		scheduler.runDailyReport()
		return
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go scheduler.Start()

	log.Println("CoinDaily 已启动，按 Ctrl+C 退出")
	<-sigChan

	log.Println("收到停止信号，正在关闭...")
	scheduler.Stop()
	log.Println("CoinDaily 已停止")
}
