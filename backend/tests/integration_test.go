package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"spendsense-backend/internal/auth"
	"spendsense-backend/internal/budget"
	"spendsense-backend/internal/category"
	"spendsense-backend/internal/expense"
	"spendsense-backend/internal/income"
	"spendsense-backend/internal/infra"
	"spendsense-backend/internal/notification"
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
	incomeSvc := income.NewService(income.NewRepository(db), wallet.NewRepository(db), categorySvc, nil)

	uniqueEmail := fmt.Sprintf("integration-%s@example.com", uuid.NewString())
	registerResp, err := authSvc.Register(ctx, auth.RegisterRequest{
		Email:    uniqueEmail,
		Password: "StrongPass123!",
	}, auth.SessionMetadata{})
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

	categories, err := categorySvc.ListCategories(ctx, userID, category.KindIncome)
	if err != nil {
		t.Fatalf("list categories failed: %v", err)
	}
	if len(categories) == 0 {
		t.Fatalf("expected at least one income category")
	}

	var incomeCategoryID uuid.UUID
	for _, item := range categories {
		if item.Kind == category.KindIncome {
			incomeCategoryID = item.ID
			break
		}
	}
	if incomeCategoryID == uuid.Nil {
		t.Fatalf("expected an income category")
	}

	incomeDate := time.Now().Add(-1 * time.Hour)
	createdIncome, err := incomeSvc.CreateIncome(ctx, userID, income.CreateRequest{
		WalletID:   createdWallet.ID,
		CategoryID: &incomeCategoryID,
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

func TestBudgetAlertAutoUpdateAndClear(t *testing.T) {
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
	budgetSvc := budget.NewService(budget.NewRepository(db), nil)
	expenseSvc := expense.NewService(expense.NewRepository(db), wallet.NewRepository(db), categorySvc, nil)
	notificationSvc := notification.NewService(db)

	// Register user
	uniqueEmail := fmt.Sprintf("integration-budget-%s@example.com", uuid.NewString())
	registerResp, err := authSvc.Register(ctx, auth.RegisterRequest{
		Email:    uniqueEmail,
		Password: "StrongPass123!",
	}, auth.SessionMetadata{})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	userID := registerResp.User.ID

	// Create wallet
	createdWallet, err := walletSvc.CreateWallet(ctx, userID, wallet.CreateRequest{
		Name:           "Main Wallet",
		WalletType:     "CASH",
		Currency:       "USD",
		OpeningBalance: 1000,
	})
	if err != nil {
		t.Fatalf("create wallet failed: %v", err)
	}

	// Get an expense category (e.g. Food)
	categories, err := categorySvc.ListCategories(ctx, userID, category.KindExpense)
	if err != nil {
		t.Fatalf("list categories failed: %v", err)
	}
	var expenseCategoryID uuid.UUID
	for _, item := range categories {
		if item.Name == "Food" {
			expenseCategoryID = item.ID
			break
		}
	}
	if expenseCategoryID == uuid.Nil && len(categories) > 0 {
		expenseCategoryID = categories[0].ID
	}
	if expenseCategoryID == uuid.Nil {
		t.Fatalf("expected an expense category")
	}

	// Create budget for category with limit $100
	createdBudget, err := budgetSvc.Create(ctx, userID, budget.CreateRequest{
		CategoryID:      expenseCategoryID,
		Amount:          100,
		Currency:        "USD",
		Period:          "MONTHLY",
		RolloverEnabled: false,
	})
	if err != nil {
		t.Fatalf("create budget failed: %v", err)
	}

	// 1. Initial State Check: No notifications
	err = notificationSvc.RunChecks(ctx, userID)
	if err != nil {
		t.Fatalf("run checks failed: %v", err)
	}
	notifs, err := notificationSvc.List(ctx, userID, 10, false)
	if err != nil {
		t.Fatalf("list notifications failed: %v", err)
	}
	if len(notifs) != 0 {
		t.Fatalf("expected 0 notifications, got %d", len(notifs))
	}

	// 2. Create expense of $80 (80% of budget, triggering 75% threshold alert)
	exp1, err := expenseSvc.CreateExpense(ctx, userID, expense.CreateRequest{
		WalletID:   createdWallet.ID,
		CategoryID: expenseCategoryID,
		Amount:     80,
		Currency:   "USD",
		Date:       time.Now(),
	})
	if err != nil {
		t.Fatalf("create expense failed: %v", err)
	}

	err = notificationSvc.RunChecks(ctx, userID)
	if err != nil {
		t.Fatalf("run checks failed: %v", err)
	}

	notifs, err = notificationSvc.List(ctx, userID, 10, false)
	if err != nil {
		t.Fatalf("list notifications failed: %v", err)
	}
	if len(notifs) != 1 {
		t.Fatalf("expected 1 notification (75%% threshold), got %d", len(notifs))
	}
	if notifs[0].Type != "budget_alert" {
		t.Fatalf("expected notification type to be budget_alert, got %s", notifs[0].Type)
	}

	// 3. Update expense to $50 (50% of budget, falls below 75% threshold)
	_, err = expenseSvc.UpdateExpense(ctx, userID, exp1.ID, expense.UpdateRequest{
		WalletID:   createdWallet.ID,
		CategoryID: expenseCategoryID,
		Amount:     50,
		Currency:   "USD",
		Date:       time.Now(),
	})
	if err != nil {
		t.Fatalf("update expense failed: %v", err)
	}

	err = notificationSvc.RunChecks(ctx, userID)
	if err != nil {
		t.Fatalf("run checks failed: %v", err)
	}

	notifs, err = notificationSvc.List(ctx, userID, 10, false)
	if err != nil {
		t.Fatalf("list notifications failed: %v", err)
	}
	if len(notifs) != 0 {
		t.Fatalf("expected budget alert notification to be cleared when spending fell below threshold, got %d", len(notifs))
	}

	// 4. Update expense to $110 (110% of budget, crossing 75%, 90%, and 100% thresholds)
	_, err = expenseSvc.UpdateExpense(ctx, userID, exp1.ID, expense.UpdateRequest{
		WalletID:   createdWallet.ID,
		CategoryID: expenseCategoryID,
		Amount:     110,
		Currency:   "USD",
		Date:       time.Now(),
	})
	if err != nil {
		t.Fatalf("update expense failed: %v", err)
	}

	err = notificationSvc.RunChecks(ctx, userID)
	if err != nil {
		t.Fatalf("run checks failed: %v", err)
	}

	notifs, err = notificationSvc.List(ctx, userID, 10, false)
	if err != nil {
		t.Fatalf("list notifications failed: %v", err)
	}
	if len(notifs) != 3 {
		t.Fatalf("expected 3 notifications (75%%, 90%%, 100%% thresholds), got %d", len(notifs))
	}

	// 5. Delete expense (falls back to 0%)
	err = expenseSvc.SoftDeleteExpense(ctx, userID, exp1.ID)
	if err != nil {
		t.Fatalf("delete expense failed: %v", err)
	}

	err = notificationSvc.RunChecks(ctx, userID)
	if err != nil {
		t.Fatalf("run checks failed: %v", err)
	}

	notifs, err = notificationSvc.List(ctx, userID, 10, false)
	if err != nil {
		t.Fatalf("list notifications failed: %v", err)
	}
	if len(notifs) != 0 {
		t.Fatalf("expected all budget alert notifications to be cleared when expense was deleted, got %d", len(notifs))
	}

	// 6. Create a fresh expense of $95 (95% of budget, triggers 75% and 90% alerts again)
	_, err = expenseSvc.CreateExpense(ctx, userID, expense.CreateRequest{
		WalletID:   createdWallet.ID,
		CategoryID: expenseCategoryID,
		Amount:     95,
		Currency:   "USD",
		Date:       time.Now(),
	})
	if err != nil {
		t.Fatalf("create expense failed: %v", err)
	}

	err = notificationSvc.RunChecks(ctx, userID)
	if err != nil {
		t.Fatalf("run checks failed: %v", err)
	}

	notifs, err = notificationSvc.List(ctx, userID, 10, false)
	if err != nil {
		t.Fatalf("list notifications failed: %v", err)
	}
	if len(notifs) != 2 {
		t.Fatalf("expected 2 notifications (75%% and 90%%) to trigger successfully after clearing, got %d", len(notifs))
	}

	// Clean up budget
	err = budgetSvc.Delete(ctx, userID, createdBudget.ID)
	if err != nil {
		t.Fatalf("failed to delete budget: %v", err)
	}
}

