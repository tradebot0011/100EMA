# Railway Go Trading Bot (100 EMA Strategy)

This project is a 24/7 automated Go trading signal bot tailored for deployment on Railway.

## Tracked Assets
- **Gold**: `XAU/USD`
- **Crude Oil**: `WTI/USD`
- **US 30 (Dow)**: `DJI`
- **Nikkei 225**: `N225`

## Strategy Rules (10m Timeframe)
- **100 EMA Monitoring**: Calculates the 100-period Exponential Moving Average recursively.
- **Buy Trigger**: 2 consecutive candle closes **above** the 100 EMA.
- **Sell Trigger**: 2 consecutive candle closes **below** the 100 EMA.
- **Stop Loss**: Default fixed at **0.25%** from entry price, or triggered when 2 consecutive candles close on the opposite side of the 100 EMA.

## Deployment Setup on Railway

1. Push these files to a GitHub repository.
2. Link the repository to a new project in [Railway.app](https://railway.app).
3. In your Railway service dashboard, set the environment variables:
   - `TELEGRAM_BOT_TOKEN`: Token from `@BotFather`
   - `TELEGRAM_CHAT_ID`: Your chat ID from `@userinfobot`
   - `MARKET_DATA_API_KEY`: API Key from [Twelve Data](https://twelvedata.com)
