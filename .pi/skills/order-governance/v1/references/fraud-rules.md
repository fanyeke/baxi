# Fraud Detection Rules Reference

## High-Risk Indicators
- Email: disposable domains (tempail.com, guerrillamail.com)
- Phone: VoIP numbers (Google Voice, TextNow)
- Address: freight forwarders, known fraud addresses
- BIN: high-risk bank identification numbers

## Velocity Rules
- Same IP: max 3 orders per hour
- Same device: max 5 orders per day
- Same email: max 2 orders per hour
- Same card: max 1 order per 10 minutes

## Category Risk Levels
| Category | Risk Level |
|----------|------------|
| Electronics | HIGH |
| Luxury | HIGH |
| Digital | MEDIUM |
| Clothing | LOW |
| Home | LOW |

## Thresholds
| Parameter | Default | Description |
|-----------|---------|-------------|
| AOV_MULTIPLIER | 3.0 | Flag if order > N × AOV |
| NEW_ACCOUNT_DAYS | 7 | Account age threshold |
| HIGH_VALUE_ORDER | $500 | Require review above this |
| VELOCITY_LIMIT | 3 | Max orders per IP per hour |
