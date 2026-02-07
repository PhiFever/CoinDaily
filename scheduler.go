package main

import (
	"fmt"
	"log"
	"time"
)

type Scheduler struct {
	config        *Config
	coinClient    *CoinGeckoClient
	emailSender   *EmailSender
	discordSender *DiscordSender
	reportGen     *ReportGenerator
	stopChan      chan bool
}

func NewScheduler(config *Config) *Scheduler {
	scheduler := &Scheduler{
		config:     config,
		coinClient: NewCoinGeckoClient(config.CoinGecko.APIKey, config.Proxy.Enabled, config.Proxy.URL),
		reportGen:  NewReportGenerator(),
		stopChan:   make(chan bool),
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
	log.Println("开始生成每日加密货币价格报表...")

	coins, err := s.coinClient.GetCoinPrices(s.config.Coins)
	if err != nil {
		log.Printf("获取加密货币价格失败: %v", err)
		return
	}

	if len(coins) == 0 {
		log.Println("未获取到任何加密货币数据")
		return
	}

	log.Printf("成功获取到 %d 个加密货币的价格数据", len(coins))

	// 记录发送结果
	emailSuccess := false
	discordSuccess := false

	// 发送邮件报表（如果配置了邮件）
	if s.emailSender != nil && s.emailSender.IsConfigured() {
		htmlReport := s.reportGen.GenerateHTMLReport(coins)
		subject := fmt.Sprintf("每日加密货币价格报表 - %s", time.Now().Format("2006年01月02日"))

		err = s.emailSender.SendReport(subject, htmlReport)
		if err != nil {
			log.Printf("发送邮件失败: %v", err)
		} else {
			log.Println("每日报表已成功发送到邮箱")
			emailSuccess = true
		}
	}

	// 发送 Discord 报表（如果配置了 Discord）
	if s.discordSender != nil && s.discordSender.IsConfigured() {
		err = s.discordSender.SendReport(coins)
		if err != nil {
			log.Printf("发送 Discord 消息失败: %v", err)
		} else {
			log.Println("每日报表已成功发送到 Discord")
			discordSuccess = true
		}
	}

	// 检查是否有任何通知渠道配置
	hasEmail := s.emailSender != nil && s.emailSender.IsConfigured()
	hasDiscord := s.discordSender != nil && s.discordSender.IsConfigured()

	if !hasEmail && !hasDiscord {
		log.Println("警告: 没有配置任何通知渠道（邮件或 Discord）")
		return
	}

	// 汇总发送结果
	if hasEmail && hasDiscord {
		if emailSuccess && discordSuccess {
			log.Println("所有通知渠道发送成功")
		} else if emailSuccess {
			log.Println("邮件发送成功，Discord 发送失败")
		} else if discordSuccess {
			log.Println("Discord 发送成功，邮件发送失败")
		} else {
			log.Println("所有通知渠道发送失败")
		}
	}
}
