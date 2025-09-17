# eBay Developer App ID Setup Guide

## Step-by-Step Registration

### 1. Create eBay Developer Account (5 min)
1. Visit [eBay Developers Program](https://developer.ebay.com/)
2. Click "Get Started" or "Join"
3. Sign in with existing eBay account or create new one
4. Accept developer terms and conditions

### 2. Create Application (10 min)
1. Navigate to "My Account" → "Applications"
2. Click "Create Application"
3. Fill out application details:
   - **Application Title**: "Pokemon Card Price Analyzer"
   - **Application Description**: "Tool for analyzing Pokemon card grading opportunities"
   - **Application Type**: "Personal/Educational"
   - **Platform**: "Desktop Application"

### 3. Get Application Keys (2 min)
1. Once approved, click on your application
2. Copy the **App ID** (also called Client ID)
3. Note: Client Secret not needed for Finding API

### 4. Configure Environment (1 min)
```bash
export EBAY_APP_ID="your_app_id_here"
# Add to ~/.bashrc or ~/.zshrc for persistence
echo 'export EBAY_APP_ID="your_app_id_here"' >> ~/.bashrc
```

### 5. Test Integration (2 min)
```bash
./pkmgradegap --set "Surging Sparks" --with-ebay --ebay-max 2 --top 1
```

## Rate Limits & Usage

### Free Tier Limits
- **5,000 calls per day**
- **Rate limit**: ~10 calls per second
- **No authentication** required for Finding API
- **Production ready** for small to medium usage

### Monitoring Usage
The application automatically handles rate limiting with exponential backoff.

### Upgrading if Needed
- **Developer tier**: 100,000 calls/day ($0.05 per additional 1,000)
- **Commercial tier**: Custom pricing for high volume

## Troubleshooting

### Common Issues
1. **HTTP 500 Error**: Invalid App ID, check credentials
2. **HTTP 429 Error**: Rate limit exceeded, wait 1 hour
3. **No Results**: eBay query too specific, try broader search terms
4. **SSL Errors**: Update system certificates

### Support Resources
- [eBay Developer Forums](https://developer.ebay.com/support)
- [Finding API Documentation](https://developer.ebay.com/api-docs/buy/browse/overview.html)
- [Application Dashboard](https://developer.ebay.com/my/keys)

## Privacy & Security

### Data Usage
- Only searches public eBay listings
- No personal information transmitted
- App ID can be safely shared (read-only access)

### Best Practices
- Don't commit App ID to public repositories
- Use environment variables for configuration
- Monitor usage in eBay Developer dashboard

## Alternative Options

### If eBay Registration Issues
1. **Use Mock Mode**: `EBAY_APP_ID="mock"`
2. **Manual Research**: Search eBay manually and compare prices
3. **Community Contribution**: Share your findings with the project

### For High-Volume Users
Consider implementing additional price sources:
- TCGPlayer direct integration
- COMC (Check Out My Cards) API
- Pokemon Center Store API

## Testing Without Real eBay App ID

For testing without real eBay App ID:
```bash
export EBAY_APP_ID="mock"
./pkmgradegap --set "Surging Sparks" --with-ebay --ebay-max 3
```

Mock mode provides realistic sample listings for demonstration purposes. The mock data includes:
- Realistic pricing based on card popularity
- Various condition grades
- Proper listing titles and URLs
- Simulated bid activity

### Mock Mode Features
- **Intelligent Pricing**: Popular Pokemon (Charizard, Pikachu, etc.) get higher base prices
- **Rarity Adjustment**: Lower card numbers typically priced higher
- **Condition Variety**: Mix of Near Mint, Lightly Played, and other conditions
- **Auction Simulation**: Includes both Buy It Now and auction-style listings
- **Realistic URLs**: Mock eBay listing URLs for testing

This allows you to test the full eBay integration features without needing a real eBay Developer account.