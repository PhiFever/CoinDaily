package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

type EmailConfig struct {
	SMTPServer   string
	SMTPPort     int
	Username     string
	Password     string
	To           []string
	ProxyEnabled bool
	ProxyURL     string
}

type EmailSender struct {
	config EmailConfig
}

func NewEmailSender(config EmailConfig) *EmailSender {
	return &EmailSender{
		config: config,
	}
}

// dialWithProxy 通过 HTTP 代理建立 TCP 连接（HTTP CONNECT 隧道）
func (e *EmailSender) dialWithProxy(targetAddr string) (net.Conn, error) {
	proxyURL, err := url.Parse(e.config.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	proxyAddr := proxyURL.Host
	if proxyURL.Port() == "" {
		proxyAddr = proxyURL.Hostname() + ":80"
	}

	// 连接到代理服务器
	conn, err := net.DialTimeout("tcp", proxyAddr, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %w", err)
	}

	// 发送 HTTP CONNECT 请求
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", targetAddr, targetAddr)
	if _, err := conn.Write([]byte(connectReq)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send CONNECT request: %w", err)
	}

	// 读取代理响应
	reader := bufio.NewReader(conn)
	resp, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read proxy response: %w", err)
	}

	// 检查响应状态
	if !strings.Contains(resp, "200") {
		conn.Close()
		return nil, fmt.Errorf("proxy CONNECT failed: %s", strings.TrimSpace(resp))
	}

	// 读取剩余的响应头（直到空行）
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to read proxy headers: %w", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	return conn, nil
}

func (e *EmailSender) SendReport(subject string, htmlContent string) error {
	from := e.config.Username
	to := e.config.To

	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(to, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=utf-8"
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + htmlContent

	addr := fmt.Sprintf("%s:%d", e.config.SMTPServer, e.config.SMTPPort)

	// 根据是否启用代理选择连接方式
	if e.config.ProxyEnabled && e.config.ProxyURL != "" {
		return e.sendWithProxy(addr, from, to, []byte(message))
	}

	// 直连模式
	auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.SMTPServer)
	err := smtp.SendMail(addr, auth, from, to, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// sendWithProxy 通过代理发送邮件
func (e *EmailSender) sendWithProxy(addr, from string, to []string, msg []byte) error {
	// 通过代理建立连接
	conn, err := e.dialWithProxy(addr)
	if err != nil {
		return fmt.Errorf("failed to dial via proxy: %w", err)
	}
	defer conn.Close()

	// 设置连接超时
	conn.SetDeadline(time.Now().Add(2 * time.Minute))

	// 创建 SMTP 客户端
	client, err := smtp.NewClient(conn, e.config.SMTPServer)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// 发送 EHLO
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("EHLO failed: %w", err)
	}

	// 启用 STARTTLS
	tlsConfig := &tls.Config{
		ServerName: e.config.SMTPServer,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	// 认证
	auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.SMTPServer)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// 设置发件人
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// 设置收件人
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("RCPT TO failed: %w", err)
		}
	}

	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}
	if _, err := writer.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	// 退出
	if err := client.Quit(); err != nil {
		// Quit 错误通常可以忽略，邮件已发送
		return nil
	}

	return nil
}
