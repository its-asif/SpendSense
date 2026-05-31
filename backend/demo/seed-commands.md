# Dashboard Demo Data (Data-Only Seeding)

This uses only:
- one JSON file: `backend/demo/long-profile-seed.json`
- terminal commands (`curl` + `jq`)

No backend code changes are required to use this.

## 1) Prerequisites

```bash
cd backend
command -v jq
command -v curl
```

## 2) Set variables

```bash
export API_URL="http://localhost:8080"
export SEED_FILE="demo/long-profile-seed.json"
export EMAIL="$(jq -r '.user.email' "$SEED_FILE")"
export PASSWORD="$(jq -r '.user.password' "$SEED_FILE")"
```

## 3) Create demo user

If user already exists, login will still work.

```bash
curl -sS -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "$(jq -c '.user | {email, password}' "$SEED_FILE")"
```

## 4) Login and save access token

```bash
export ACCESS_TOKEN="$(curl -sS -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "$(jq -nc --arg email "$EMAIL" --arg password "$PASSWORD" '{email:$email,password:$password}')" \
  | jq -r '.access_token')"

echo "$ACCESS_TOKEN" | head -c 24; echo
```

## 5) Use the built-in categories

The backend currently returns `500` for `POST /api/v1/categories`, so this demo uses the default categories that are already seeded by the app.

```bash
curl -sS "$API_URL/api/v1/categories" -H "Authorization: Bearer $ACCESS_TOKEN" > /tmp/demo_categories.json
```

## 6) Create wallets (from JSON)

```bash
jq -c '.wallets[] | del(.key)' "$SEED_FILE" | while read -r row; do
  curl -sS -X POST "$API_URL/api/v1/wallets" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$row" >/dev/null

  echo "created wallet: $(echo "$row" | jq -r '.name')"
done
```

## 7) Top up the card wallet for the longer expense run

The demo spends heavily from `Daily Card`, so add a transfer from `Main Checking` before creating the remaining expenses.

```bash
MAIN_WALLET_ID="$(jq -r '.wallets[] | select(.name=="Main Checking") | .id' /tmp/demo_wallets.json | head -n1)"
CARD_WALLET_ID="$(jq -r '.wallets[] | select(.name=="Daily Card") | .id' /tmp/demo_wallets.json | head -n1)"

curl -sS -X POST "$API_URL/api/v1/wallets/transfer" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "$(jq -nc --arg from_wallet_id "$MAIN_WALLET_ID" --arg to_wallet_id "$CARD_WALLET_ID" --arg currency "USD" '{from_wallet_id:$from_wallet_id,to_wallet_id:$to_wallet_id,amount:5000,exchange_rate:1,fee_amount:0,currency:$currency,transfer_date:"2026-05-31",notes:"demo top up"}')" >/dev/null
```

## 8) Cache category/wallet lists for ID lookup

```bash
curl -sS "$API_URL/api/v1/categories" -H "Authorization: Bearer $ACCESS_TOKEN" > /tmp/demo_categories.json
curl -sS "$API_URL/api/v1/wallets" -H "Authorization: Bearer $ACCESS_TOKEN" > /tmp/demo_wallets.json
```

## 9) Create incomes across long date range

```bash
map_category_name() {
  case "$1" in
    Housing|Food|Transport|Utilities|Entertainment|Shopping|Health)
      echo "$1"
      ;;
    *)
      echo "Other"
      ;;
  esac
}

jq -c '.incomes[]' "$SEED_FILE" | while read -r row; do
  wallet_key="$(echo "$row" | jq -r '.wallet_key')"
  wallet_name="$(jq -r --arg key "$wallet_key" '.wallets[] | select(.key==$key) | .name' "$SEED_FILE")"
  category_name="$(echo "$row" | jq -r '.category_name')"

  wallet_id="$(jq -r --arg n "$wallet_name" '.wallets[] | select(.name==$n) | .id' /tmp/demo_wallets.json | head -n1)"
  mapped_category_name="$(map_category_name "$category_name")"
  category_id="$(jq -r --arg n "$mapped_category_name" '.categories[] | select(.name==$n) | .id' /tmp/demo_categories.json | head -n1)"

  payload="$(echo "$row" | jq -c --arg wallet_id "$wallet_id" --arg category_id "$category_id" '
    {
      wallet_id: $wallet_id,
      category_id: ($category_id | if .=="" then null else . end),
      source_name, amount, currency, income_date, notes
    }
  ')"

  curl -sS -X POST "$API_URL/api/v1/incomes" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$payload" >/dev/null

  echo "created income: $(echo "$row" | jq -r '.income_date + " | " + .source_name + " | " + (.amount|tostring)')"
done
```

## 10) Create expenses across long date range

```bash
map_category_name() {
  case "$1" in
    Housing|Food|Transport|Utilities|Entertainment|Shopping|Health)
      echo "$1"
      ;;
    Pet\ Care|Fitness)
      echo "Health"
      ;;
    *)
      echo "Other"
      ;;
  esac
}

jq -c '.expenses[]' "$SEED_FILE" | while read -r row; do
  wallet_key="$(echo "$row" | jq -r '.wallet_key')"
  wallet_name="$(jq -r --arg key "$wallet_key" '.wallets[] | select(.key==$key) | .name' "$SEED_FILE")"
  category_name="$(echo "$row" | jq -r '.category_name')"

  wallet_id="$(jq -r --arg n "$wallet_name" '.wallets[] | select(.name==$n) | .id' /tmp/demo_wallets.json | head -n1)"
  mapped_category_name="$(map_category_name "$category_name")"
  category_id="$(jq -r --arg n "$mapped_category_name" '.categories[] | select(.name==$n) | .id' /tmp/demo_categories.json | head -n1)"

  payload="$(echo "$row" | jq -c --arg wallet_id "$wallet_id" --arg category_id "$category_id" '
    {
      wallet_id: $wallet_id,
      amount, currency, fx_rate_to_base,
      category_id: $category_id,
      merchant, date, notes,
      is_recurring: false
    }
  ')"

  curl -sS -X POST "$API_URL/api/v1/expenses" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$payload" >/dev/null

  echo "created expense: $(echo "$row" | jq -r '.date + " | " + .category_name + " | " + (.amount|tostring)')"
done
```

## 10) Quick checks

```bash
curl -sS "$API_URL/api/v1/expenses?limit=5" -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.expenses | length'
curl -sS "$API_URL/api/v1/incomes?limit=5" -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.incomes | length'
curl -sS "$API_URL/api/v1/dashboard/summary" -H "Authorization: Bearer $ACCESS_TOKEN" | jq '{monthly_income,monthly_expenses,total_balance}'
```

## Optional: reset this demo user and reseed

```bash
# Re-run from step 3 with same email/password.
# The dataset spans 2025-06 through 2026-05 for graph/history visualization.
```
