package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"spendsense-backend/internal/auth"
	"spendsense-backend/internal/category"
	"spendsense-backend/internal/income"
	"spendsense-backend/internal/infra"
	"spendsense-backend/internal/wallet"

	"github.com/google/uuid"
)

func TestAuthAndIncomeFlow(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://spendsense:spendsense@localhost:5432/spendsense?sslmode=disable"
	}

	db, err := infra.NewDatabase(databaseURL)
	if err != nil {
		t.Skipf("skipping integration test; database is unavailable: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	authSvc := auth.NewAuthService(db, auth.NewJWTManager("integration-test-secret"))
	walletSvc := wallet.NewService(wallet.NewRepository(db), nil)
	categorySvc := category.NewService(category.NewRepository(db))
	incomeSvc := income.NewService(income.NewRepository(db), wallet.NewRepository(db), nil)

	uniqueEmail := fmt.Sprintf("integration-%s@example.com", uuid.NewString())
	registerResp, err := authSvc.Register(ctx, auth.RegisterRequest{
		Email:    uniqueEmail,
		Password: "StrongPass123!",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if registerResp.AccessToken == "" || registerResp.RefreshToken == "" {
		t.Fatalf("expected register response to include access and refresh tokens")
	}

	userID := registerResp.User.ID

	createdWallet, err := walletSvc.CreateWallet(ctx, userID, wallet.CreateRequest{
		Name:           "Primary Wallet",
		WalletType:     "CASH",
		Currency:       "USD",
		OpeningBalance: 1000,
	})
	if err != nil {
		t.Fatalf("create wallet failed: %v", err)
	}

	categories, err := categorySvc.ListCategories(ctx, userID)
	if err != nil {
		t.Fatalf("list categories failed: %v", err)
	}
	if len(categories) == 0 {
		t.Fatalf("expected at least one category")
	}

	incomeDate := time.Now().Add(-1 * time.Hour)
	createdIncome, err := incomeSvc.CreateIncome(ctx, userID, income.CreateRequest{
		WalletID:   createdWallet.ID,
		CategoryID: &categories[0].ID,
		SourceName: "Salary",
		Amount:     2500,
		Currency:   "usd",
		IncomeDate: incomeDate,
	})
	if err != nil {
		t.Fatalf("create income failed: %v", err)
	}

	if createdIncome.ID == uuid.Nil {
		t.Fatalf("expected created income ID")
	}

	updatedWallet, err := walletSvc.GetWallet(ctx, userID, createdWallet.ID)
	if err != nil {
		t.Fatalf("get wallet after income create failed: %v", err)
	}
	if updatedWallet.CurrentBalance != 3500 {
		t.Fatalf("expected wallet balance to increase to 3500, got %v", updatedWallet.CurrentBalance)
	}

	if err := incomeSvc.SoftDeleteIncome(ctx, userID, createdIncome.ID); err != nil {
		t.Fatalf("delete income failed: %v", err)
	}

	restoredWallet, err := walletSvc.GetWallet(ctx, userID, createdWallet.ID)
	if err != nil {
		t.Fatalf("get wallet after income delete failed: %v", err)
	}
	if restoredWallet.CurrentBalance != 1000 {
		t.Fatalf("expected wallet balance to restore to 1000, got %v", restoredWallet.CurrentBalance)
	}

	incomes, nextPagination, err := incomeSvc.ListIncomes(ctx, userID, 20, "")
	if err != nil {
		t.Fatalf("list incomes failed: %v", err)
	}
	if nextPagination != "" && len(incomes) == 0 {
		t.Fatalf("unexpected non-empty next pagination with empty list")
	}

	found := false
	for _, item := range incomes {
		if item.ID == createdIncome.ID {
			found = true
			break
		}
	}
	if found {
		t.Fatalf("deleted income should not be returned in list response")
	}
}
