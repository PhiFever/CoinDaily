package main

import (
	"fmt"
	"strings"
	"time"
)

type ReportGenerator struct{}

func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{}
}

func (r *ReportGenerator) GenerateHTMLReport(coins []CoinPrice) string {
	now := time.Now()
	dateStr := now.Format("2006年01月02日")
	
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>每日加密货币价格报表 - %s</title>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            margin: 20px; 
            background-color: #f5f5f5;
        }
        .header { 
            text-align: center; 
            color: #2c3e50; 
            margin-bottom: 30px;
            padding: 20px;
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .report-date { 
            color: #7f8c8d; 
            font-size: 16px; 
            margin-top: 10px;
        }
        table { 
            width: 100%%; 
            border-collapse: collapse; 
            background-color: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        th, td { 
            padding: 12px; 
            text-align: left; 
            border-bottom: 1px solid #ecf0f1;
        }
        th { 
            background-color: #34495e; 
            color: white; 
            font-weight: bold;
        }
        tr:hover { 
            background-color: #f8f9fa; 
        }
        .positive { 
            color: #27ae60; 
            font-weight: bold;
        }
        .negative { 
            color: #e74c3c; 
            font-weight: bold;
        }
        .price { 
            font-weight: bold; 
            font-size: 16px;
        }
        .footer { 
            text-align: center; 
            margin-top: 30px; 
            color: #7f8c8d; 
            font-size: 14px;
            padding: 15px;
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 每日加密货币价格报表</h1>
        <div class="report-date">%s</div>
    </div>
    
    <table>
        <thead>
            <tr>
                <th>币种</th>
                <th>符号</th>
                <th>当前价格 (USD)</th>
                <th>24h 变化</th>
                <th>24h 变化率</th>
                <th>市值</th>
                <th>24h 交易量</th>
            </tr>
        </thead>
        <tbody>`, dateStr, dateStr)

	for _, coin := range coins {
		changeClass := "positive"
		changeSymbol := "+"
		if coin.PriceChange24h < 0 {
			changeClass = "negative"
			changeSymbol = ""
		}
		
		percChangeClass := "positive"
		percChangeSymbol := "+"
		if coin.PriceChangePerc24h < 0 {
			percChangeClass = "negative"
			percChangeSymbol = ""
		}

		html += fmt.Sprintf(`
            <tr>
                <td><strong>%s</strong></td>
                <td>%s</td>
                <td class="price">$%s</td>
                <td class="%s">%s$%s</td>
                <td class="%s">%s%.2f%%</td>
                <td>$%s</td>
                <td>$%s</td>
            </tr>`,
			coin.Name,
			strings.ToUpper(coin.Symbol),
			formatNumber(coin.CurrentPrice),
			changeClass, changeSymbol, formatNumber(coin.PriceChange24h),
			percChangeClass, percChangeSymbol, coin.PriceChangePerc24h,
			formatLargeNumber(coin.MarketCap),
			formatLargeNumber(coin.Volume24h),
		)
	}

	html += `
        </tbody>
    </table>
    
    <div class="footer">
        <p>数据来源: CoinGecko API</p>
        <p>此报表由 CoinDaily 自动生成</p>
    </div>
</body>
</html>`

	return html
}

func formatNumber(num float64) string {
	if num >= 1 {
		return fmt.Sprintf("%.2f", num)
	} else if num >= 0.01 {
		return fmt.Sprintf("%.4f", num)
	} else {
		return fmt.Sprintf("%.8f", num)
	}
}

func formatLargeNumber(num float64) string {
	if num >= 1e12 {
		return fmt.Sprintf("%.2fT", num/1e12)
	} else if num >= 1e9 {
		return fmt.Sprintf("%.2fB", num/1e9)
	} else if num >= 1e6 {
		return fmt.Sprintf("%.2fM", num/1e6)
	} else if num >= 1e3 {
		return fmt.Sprintf("%.2fK", num/1e3)
	}
	return fmt.Sprintf("%.2f", num)
}