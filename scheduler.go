package main

import (
	"log"
	"time"
)

type Scheduler struct {
	config        *Config
	coinClient    *CoinGeckoClient
	stockClient   *AlpacaClient
	perpClient    *HyperliquidClient
	stateStore    *StateStore
	emailSender   *EmailSender
	discordSender *DiscordSender
	reportGen     *ReportGenerator
	stopChan      chan bool
}

func NewScheduler(config *Config) *Scheduler {
	scheduler := &Scheduler{
		config:    config,
		reportGen: NewReportGenerator(),
		stopChan:  make(chan bool),
	}
	if len(config.Coins) > 0 {
		scheduler.coinClient = NewCoinGeckoClient(config.CoinGecko.APIKey, config.Proxy.Enabled, config.Proxy.URL)
	}
	if len(config.Stocks) > 0 {
		scheduler.stockClient = NewAlpacaClient(config.Alpaca.APIKey, config.Alpaca.SecretKey, config.Alpaca.Feed, config.Proxy.Enabled, config.Proxy.URL)
	}
	if len(config.Hyperliquid.Perpetuals) > 0 {
		scheduler.perpClient = NewHyperliquidClient(config.Proxy.Enabled, config.Proxy.URL)
		scheduler.stateStore = NewStateStore(config.statePath())
	}

	// 如果配置了邮件，初始化邮件发送器
	if isEmailConfigured(config) {
		emailConfig := EmailConfig{
			SMTPServer:   config.Email.SMTPServer,
			SMTPPort:     config.Email.SMTPPort,
			Username:     config.Email.Username,
			Password:     config.Email.Password,
			To:           config.Email.To,
			ProxyEnabled: config.Proxy.Enabled,
			ProxyURL:     config.Proxy.URL,
		}
		scheduler.emailSender = NewEmailSender(emailConfig)
	}

	// 如果配置了 Discord，初始化 Discord 发送器
	if isDiscordConfigured(config) {
		scheduler.discordSender = NewDiscordSender(
			config.Discord.BotToken,
			config.Discord.ChannelID,
			config.Proxy.Enabled,
			config.Proxy.URL,
		)
	}

	return scheduler
}

func (s *Scheduler) Start() {
	log.Println("启动定时任务调度器...")

	s.runOnceNow()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			if now.Hour() == s.config.Schedule.Hour && now.Minute() == s.config.Schedule.Minute {
				s.runDailyReport()
			}
		case <-s.stopChan:
			log.Println("定时任务调度器已停止")
			return
		}
	}
}

func (s *Scheduler) Stop() {
	s.stopChan <- true
}

func (s *Scheduler) runOnceNow() {
	log.Println("立即执行一次报表生成...")
	s.runDailyReport()
}

func (s *Scheduler) runDailyReport() {
	log.Println("开始生成市场行情报表...")
	runner := &ReportRunner{
		CoinIDs:          s.config.Coins,
		StockSymbols:     s.config.Stocks,
		PerpetualSymbols: s.config.Hyperliquid.Perpetuals,
		StockFeed:        s.config.Alpaca.Feed,
		StateStore:       s.stateStore,
		ReportGenerator:  s.reportGen,
	}
	if s.coinClient != nil {
		runner.CryptoCollector = s.coinClient
	}
	if s.stockClient != nil {
		runner.StockCollector = s.stockClient
	}
	if s.perpClient != nil {
		runner.PerpCollector = s.perpClient
	}
	if s.emailSender != nil && s.emailSender.IsConfigured() {
		runner.EmailSender = s.emailSender
	}
	if s.discordSender != nil && s.discordSender.IsConfigured() {
		runner.DiscordSender = s.discordSender
	}

	report, err := runner.Run()
	if err != nil {
		log.Printf("市场行情报表执行存在错误: %v", err)
	}
	if report == nil || !report.HasData() {
		log.Println("所有已配置行情源均无可用数据，未发送通知")
		return
	}
	log.Printf("市场行情报表完成：加密货币 %d，美股/ETF %d，永续合约 %d", len(report.Crypto.Items), len(report.Stocks.Items), len(report.Perpetuals.Items))
}
