package main

import (
	"fmt"
	"log"
	"time"
)

type Scheduler struct {
	config      *Config
	coinClient  *CoinGeckoClient
	emailSender *EmailSender
	reportGen   *ReportGenerator
	stopChan    chan bool
}

func NewScheduler(config *Config) *Scheduler {
	emailConfig := EmailConfig{
		SMTPServer: config.Email.SMTPServer,
		SMTPPort:   config.Email.SMTPPort,
		Username:   config.Email.Username,
		Password:   config.Email.Password,
		To:         config.Email.To,
	}

	return &Scheduler{
		config:      config,
		coinClient:  NewCoinGeckoClient(config.CoinGecko.APIKey),
		emailSender: NewEmailSender(emailConfig),
		reportGen:   NewReportGenerator(),
		stopChan:    make(chan bool),
	}
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

	log.Printf("成功获取到 %d 个加密货币的价格数据，详细数据：%v", len(coins), coins)

	htmlReport := s.reportGen.GenerateHTMLReport(coins)
	// // 保存到本地进行测试
	// err = os.WriteFile("report.html", []byte(htmlReport), 0644)
	// if err != nil {
	// 	log.Printf("保存报表到本地失败: %v", err)
	// 	return
	// }

	subject := fmt.Sprintf("每日加密货币价格报表 - %s", time.Now().Format("2006年01月02日"))

	err = s.emailSender.SendReport(subject, htmlReport)
	if err != nil {
		log.Printf("发送邮件失败: %v", err)
		return
	}

	log.Println("每日报表已成功发送到邮箱")
}
