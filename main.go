package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Candle represents OHLCV market data
type Candle struct {
	Timestamp time.Time
	Close     float64
}

// PositionType represents current trade stance
type PositionType string

const (
	PositionNone PositionType = "NONE"
	PositionLong PositionType = "LONG"
	PositionShort PositionType = "SHORT"
)

// AssetTracker handles state and logic per instrument
type AssetTracker struct {
	Symbol           string
	Name             string
	EMA100           float64
	Position         PositionType
	EntryPrice       float64
	StopLossPrice    float64
	AboveCount       int
	BelowCount       int
	HistoricalCloses []float64
}

// Global Config
var (
	TelegramBotToken string
	TelegramChatID   int64
	Bot              *tgbotapi.BotAPI
)

func main() {
	// 1. Fetch Railway / Environment Configuration
	TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")

	if TelegramBotToken == "" || chatIDStr == "" {
		log.Fatal("Error: TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID must be set.")
	}

	var err error
	TelegramChatID, err = strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid TELEGRAM_CHAT_ID: %v", err)
	}

	Bot, err = tgbotapi.NewBotAPI(TelegramBotToken)
	if err != nil {
		log.Fatalf("Telegram initialization failed: %v", err)
	}

	log.Printf("Authorized on Telegram account: %s", Bot.Self.UserName)

	// 2. Initialize Assets (Gold, Oil, US30, Nikkei 225)
	assets := []*AssetTracker{
		{Symbol: "XAU/USD", Name: "Gold"},
		{Symbol: "WTI/USD", Name: "Crude Oil"},
		{Symbol: "DJI", Name: "Dow Jones (US 30)"},
		{Symbol: "N225", Name: "Nikkei 225"},
	}

	// Send Startup Notification
	sendTelegramMsg("🚀 **Trading Bot Started on Railway**\nMonitoring 10m timeframe for Gold, Oil, US 30, and Nikkei 225.")

	// 3. Execution Loop: Runs every 10 minutes
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	// Initial evaluation on launch
	evaluateAll(assets)

	for range ticker.C {
		evaluateAll(assets)
	}
}

func evaluateAll(assets []*AssetTracker) {
	for _, asset := range assets {
		err := processAsset(asset)
		if err != nil {
			log.Printf("[%s] Error processing strategy: %v", asset.Symbol, err)
		}
	}
}

func processAsset(asset *AssetTracker) error {
	// Fetch 10-minute candles
	candles, err := fetch10mCandles(asset.Symbol)
	if err != nil {
		return err
	}

	if len(candles) < 100 {
		return fmt.Errorf("insufficient history to calculate 100 EMA")
	}

	// Update 100 EMA with current candles
	asset.EMA100 = calculateEMA(candles, 100)
	latestCandle := candles[len(candles)-1]
	currentClose := latestCandle.Close

	log.Printf("[%s] Price: %.2f | 100 EMA: %.2f", asset.Symbol, currentClose, asset.EMA100)

	// Evaluate Stop Loss if currently in a trade
	if asset.Position == PositionLong {
		if currentClose <= asset.StopLossPrice {
			sendTelegramMsg(fmt.Sprintf("🛑 **STOP LOSS HIT (LONG)**\nAsset: %s\nExit Price: %.2f\nStop Price: %.2f", asset.Name, currentClose, asset.StopLossPrice))
			asset.Position = PositionNone
		}
	} else if asset.Position == PositionShort {
		if currentClose >= asset.StopLossPrice {
			sendTelegramMsg(fmt.Sprintf("🛑 **STOP LOSS HIT (SHORT)**\nAsset: %s\nExit Price: %.2f\nStop Price: %.2f", asset.Name, currentClose, asset.StopLossPrice))
			asset.Position = PositionNone
		}
	}

	// Evaluate 100 EMA Crossover Logic (2 candle closes)
	if currentClose > asset.EMA100 {
		asset.AboveCount++
		asset.BelowCount = 0
	} else if currentClose < asset.EMA100 {
		asset.BelowCount++
		asset.AboveCount = 0
	} else {
		asset.AboveCount = 0
		asset.BelowCount = 0
	}

	// BUY TRIGGER (2 consecutive candle closes above 100 EMA)
	if asset.AboveCount >= 2 {
		if asset.Position == PositionShort {
			sendTelegramMsg(fmt.Sprintf("🔄 **EXIT SHORT (Opposite 2-Close Signal)**\nAsset: %s\nPrice: %.2f", asset.Name, currentClose))
			asset.Position = PositionNone
		}

		if asset.Position == PositionNone {
			asset.Position = PositionLong
			asset.EntryPrice = currentClose
			asset.StopLossPrice = currentClose * (1 - 0.0025)

			msg := fmt.Sprintf("🟢 **BUY SIGNAL DETECTED**\n\n**Asset:** %s (%s)\n**Timeframe:** 10m\n**Entry Price:** %.2f\n**100 EMA:** %.2f\n**Stop Loss (0.25%%):** %.2f",
				asset.Name, asset.Symbol, asset.EntryPrice, asset.EMA100, asset.StopLossPrice)
			sendTelegramMsg(msg)
		}
	}

	// SELL TRIGGER (2 consecutive candle closes below 100 EMA)
	if asset.BelowCount >= 2 {
		if asset.Position == PositionLong {
			sendTelegramMsg(fmt.Sprintf("🔄 **EXIT LONG (Opposite 2-Close Signal)**\nAsset: %s\nPrice: %.2f", asset.Name, currentClose))
			asset.Position = PositionNone
		}

		if asset.Position == PositionNone {
			asset.Position = PositionShort
			asset.EntryPrice = currentClose
			asset.StopLossPrice = currentClose * (1 + 0.0025)

			msg := fmt.Sprintf("🔴 **SELL SIGNAL DETECTED**\n\n**Asset:** %s (%s)\n**Timeframe:** 10m\n**Entry Price:** %.2f\n**100 EMA:** %.2f\n**Stop Loss (0.25%%):** %.2f",
				asset.Name, asset.Symbol, asset.EntryPrice, asset.EMA100, asset.StopLossPrice)
			sendTelegramMsg(msg)
		}
	}

	return nil
}

func calculateEMA(candles []Candle, period int) float64 {
	k := 2.0 / float64(period+1)
	var sum float64
	for i := 0; i < period; i++ {
		sum += candles[i].Close
	}
	ema := sum / float64(period)

	for i := period; i < len(candles); i++ {
		ema = (candles[i].Close * k) + (ema * (1 - k))
	}
	return math.Round(ema*100) / 100
}

func fetch10mCandles(symbol string) ([]Candle, error) {
	apiKey := os.Getenv("MARKET_DATA_API_KEY")
	url := fmt.Sprintf("https://api.twelvedata.com/time_series?symbol=%s&interval=10min&outputsize=150&apikey=%s", symbol, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	type TwelveDataResponse struct {
		Values []struct {
			Datetime string `json:"datetime"`
			Close    string `json:"close"`
		} `json:"values"`
	}

	var data TwelveDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var candles []Candle
	for i := len(data.Values) - 1; i >= 0; i-- {
		val := data.Values[i]
		price, _ := strconv.ParseFloat(val.Close, 64)
		t, _ := time.Parse("2006-01-02 15:04:05", val.Datetime)
		candles = append(candles, Candle{Timestamp: t, Close: price})
	}

	return candles, nil
}

func sendTelegramMsg(message string) {
	msg := tgbotapi.NewMessage(TelegramChatID, message)
	msg.ParseMode = "Markdown"
	_, err := Bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send Telegram message: %v", err)
	}
}
