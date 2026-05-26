package wallet

import (
	"context"
	"errors"
	"strings"
	"time"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

func parseTransferDate(input string) (time.Time, error) {
	if input == "" {
		return time.Time{}, errors.New("empty transfer date")
	}

	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, input)
		if err != nil {
			continue
		}

		parsed = parsed.UTC()
		return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC), nil
	}

	return time.Time{}, errors.New("invalid transfer date")
}

func (s *Service) Transfer(ctx context.Context, userID uuid.UUID, req CreateTransferRequest) (*Transfer, error) {
	// basic validation
	if req.Amount <= 0 {
		return nil, domain.NewDomainError(domain.ErrInvalidAmount, "amount must be > 0", 400)
	}
	if req.FeeAmount < 0 {
		return nil, domain.NewDomainError(domain.ErrInvalidAmount, "fee_amount must be >= 0", 400)
	}
	if req.FromWalletID == req.ToWalletID {
		return nil, domain.NewDomainError(domain.ErrInvalidWallet, "from and to wallets must differ", 400)
	}

	// parse transfer date
	td, err := parseTransferDate(req.TransferDate)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrInvalidDate, "invalid transfer_date", 400)
	}

	fromWallet, err := s.repo.GetWalletByID(ctx, userID, req.FromWalletID)
	if err != nil {
		return nil, err
	}
	toWallet, err := s.repo.GetWalletByID(ctx, userID, req.ToWalletID)
	if err != nil {
		return nil, err
	}

	fromCurrency := strings.ToUpper(strings.TrimSpace(fromWallet.Currency))
	toCurrency := strings.ToUpper(strings.TrimSpace(toWallet.Currency))
	requestCurrency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if requestCurrency != "" && requestCurrency != fromCurrency {
		return nil, domain.NewDomainError(domain.ErrInvalidCurrency, "currency must match source wallet currency", 400)
	}

	amount := round2(req.Amount)
	feeAmount := round2(req.FeeAmount)

	exchangeRate := req.ExchangeRate
	convertedAmount := round2(amount)
	if fromCurrency == toCurrency {
		exchangeRate = 1
		convertedAmount = round2(amount)
	} else if exchangeRate <= 0 {
		if s.currencySvc == nil {
			return nil, domain.NewDomainError(domain.ErrInvalidAmount, "exchange_rate must be provided for cross-currency transfer", 400)
		}

		var rate float64
		convertedAmount, rate, err = s.currencySvc.Convert(ctx, amount, fromCurrency, toCurrency)
		if err != nil {
			return nil, domain.NewDomainError(domain.ErrInvalidCurrency, err.Error(), 400)
		}
		exchangeRate = rate
	} else {
		exchangeRate = round2(exchangeRate)
		convertedAmount = round2(amount * exchangeRate)
	}

	t := &Transfer{
		ID:              uuid.New(),
		UserID:          userID,
		FromWalletID:    req.FromWalletID,
		ToWalletID:      req.ToWalletID,
		Amount:          amount,
		ConvertedAmount: convertedAmount,
		ExchangeRate:    exchangeRate,
		FeeAmount:       feeAmount,
		Currency:        fromCurrency,
		TransferDate:    td,
		Notes:           req.Notes,
	}

	if err := s.repo.CreateTransfer(ctx, t); err != nil {
		return nil, err
	}

	return t, nil
}
