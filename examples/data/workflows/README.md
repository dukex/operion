# Operion Example Workflows

This directory contains functional workflow examples that demonstrate Operion's capabilities using real, public APIs.

## Functional Workflows (Using Real APIs)

### 🌤️ Weather Monitoring (`weather-monitoring.json`)
- **API**: OpenWeatherMap API (requires free API key)
- **Purpose**: Monitor weather conditions and alert when temperature exceeds thresholds
- **Schedule**: Every 3 hours
- **Features**:
  - Fetches current weather data
  - Processes temperature, humidity, wind data
  - Sends alerts for high temperatures
  - Posts reports to webhook (httpbin.org for testing)
- **Setup**: Set `OPENWEATHER_API_KEY` environment variable
- **Variables**: Configure `city` and `temperature_threshold`

### 💰 Cryptocurrency Tracker (`crypto-price-tracker.json`)
- **API**: CoinGecko API (no key required)
- **Purpose**: Track Bitcoin and Ethereum prices, detect significant changes
- **Schedule**: Every 5 minutes
- **Features**:
  - Fetches real crypto prices and market data
  - Calculates 24h price changes
  - Detects significant price movements (>5% by default)
  - Analyzes market cap data
- **Variables**: Configure `crypto_ids` and `change_threshold`

### 🔧 GitHub Repository Monitor (`github-repo-monitor.json`)
- **API**: GitHub API (no key required for public repos)
- **Purpose**: Monitor repository activity including issues, PRs, and releases
- **Schedule**: Every 6 hours
- **Features**:
  - Fetches repository statistics
  - Monitors recent issues and releases
  - Calculates activity scores
  - Alerts on high activity periods
- **Variables**: Configure `repo_owner`, `repo_name`, and `activity_threshold`

### 📰 News Aggregator (`news-aggregator.json`)
- **API**: NewsAPI (requires free API key)
- **Purpose**: Aggregate news from multiple sources and analyze trends
- **Schedule**: 3 times daily (8 AM, 2 PM, 8 PM UTC)
- **Features**:
  - Fetches top headlines by country/category
  - Aggregates technology news
  - Analyzes news sources and keywords
  - Detects breaking news patterns
- **Setup**: Set `NEWS_API_KEY` environment variable
- **Variables**: Configure `country`, `category`, and `breaking_news_threshold`

### 🌍 Data Synchronization (`data-sync-real.json`)
- **API**: REST Countries API (no key required)
- **Purpose**: Sync country data and validate data consistency
- **Schedule**: Weekly (Sundays at 2 AM UTC)
- **Features**:
  - Fetches all countries data
  - Validates data quality and consistency
  - Compares global vs regional data
  - Generates data quality scores
- **Variables**: Configure `sync_frequency` and `quality_threshold`

### 🛍️ Product Catalog Sync (`product-catalog-sync.json`)
- **API**: Fake Store API (no key required)
- **Purpose**: Sync product catalog and analyze inventory
- **Schedule**: Daily at 1 AM UTC
- **Features**:
  - Fetches product catalog and categories
  - Analyzes price trends and category distribution
  - Calculates inventory health scores
  - Identifies top-rated and expensive products
- **Variables**: Configure `health_threshold` and `sync_frequency`

## Legacy Examples (Basic Functionality)

### 💸 Bitcoin Price Tracker (`bitcoin-price.json`)
- **API**: CoinPaprika API (no key required)
- **Purpose**: Simple Bitcoin price fetching and logging
- **Schedule**: Every minute
- **Features**: Basic price fetching and transformation

## API Requirements Summary

| Workflow | API | Key Required | Rate Limits | Cost |
|----------|-----|--------------|-------------|------|
| Weather Monitoring | OpenWeatherMap | ✅ Free | 60 calls/min | Free tier available |
| Crypto Tracker | CoinGecko | ❌ | 10-30 req/min | Free |
| GitHub Monitor | GitHub | ❌ | 60 req/hour | Free for public repos |
| News Aggregator | NewsAPI | ✅ Free | 100 req/day | Free tier available |
| Data Sync | REST Countries | ❌ | None specified | Free |
| Product Catalog | Fake Store API | ❌ | None specified | Free |

## Getting Started

1. **Choose a workflow** based on your needs
2. **Set up API keys** if required (environment variables)
3. **Configure variables** in the workflow file
4. **Deploy the workflow** using Operion CLI or API
5. **Monitor execution** through logs and webhook reports

## Testing Webhooks

All workflows use `httpbin.org/post` for webhook testing. Replace with your actual webhook endpoints for production use.

## Customization

Each workflow can be customized by:
- Modifying schedule triggers
- Adjusting variable values
- Adding custom transformation logic
- Integrating with your own APIs
- Adding notification channels

## Support

For questions about these workflows, please refer to the main Operion documentation or create an issue in the repository.