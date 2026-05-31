package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newCategoryCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "category", Short: "Category commands"}
	cmd.AddCommand(newCategoryListCmd())
	return cmd
}

func newCategoryListCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			resp, err := client.ListCategories(cmd.Context())
			if err != nil {
				return err
			}

			switch strings.ToLower(strings.TrimSpace(output)) {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(resp.Categories)
			case "csv":
				w := csv.NewWriter(cmd.OutOrStdout())
				_ = w.Write([]string{"id", "name", "is_default", "created_at"})
				for _, item := range resp.Categories {
					_ = w.Write([]string{item.ID, item.Name, strconv.FormatBool(item.IsDefault), item.CreatedAt})
				}
				w.Flush()
				return w.Error()
			default:
				table := tablewriter.NewWriter(cmd.OutOrStdout())
				table.SetHeader([]string{"ID", "Name", "Default", "Created At"})
				for _, item := range resp.Categories {
					table.Append([]string{item.ID, item.Name, strconv.FormatBool(item.IsDefault), item.CreatedAt})
				}
				table.Render()
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&output, "output", "table", "Output format: table, json, csv")
	return cmd
}

func newWalletCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "wallet", Short: "Wallet commands"}
	cmd.AddCommand(newWalletListCmd())
	return cmd
}

func newWalletListCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List wallets",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			resp, err := client.ListWallets(cmd.Context())
			if err != nil {
				return err
			}

			switch strings.ToLower(strings.TrimSpace(output)) {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(resp.Wallets)
			case "csv":
				w := csv.NewWriter(cmd.OutOrStdout())
				_ = w.Write([]string{"id", "name", "wallet_type", "currency", "current_balance", "is_active"})
				for _, item := range resp.Wallets {
					_ = w.Write([]string{item.ID, item.Name, item.WalletType, item.Currency, fmt.Sprintf("%.2f", item.CurrentBalance), strconv.FormatBool(item.IsActive)})
				}
				w.Flush()
				return w.Error()
			default:
				table := tablewriter.NewWriter(cmd.OutOrStdout())
				table.SetHeader([]string{"ID", "Name", "Type", "Currency", "Balance", "Active"})
				for _, item := range resp.Wallets {
					table.Append([]string{item.ID, item.Name, item.WalletType, item.Currency, fmt.Sprintf("%.2f", item.CurrentBalance), strconv.FormatBool(item.IsActive)})
				}
				table.Render()
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&output, "output", "table", "Output format: table, json, csv")
	return cmd
}

func newIncomeCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "income", Short: "Income commands"}
	cmd.AddCommand(newIncomeListCmd())
	cmd.AddCommand(newIncomeAddCmd())
	cmd.AddCommand(newIncomeDeleteCmd())
	return cmd
}

func newIncomeListCmd() *cobra.Command {
	var (
		limit      int
		pagination string
		from       string
		to         string
		categoryID string
	)
	var output string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List incomes",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			resp, err := client.ListIncomes(cmd.Context(), limit, pagination, from, to, categoryID)
			if err != nil {
				return err
			}
			// try to resolve names for wallets/categories
			categories, err := client.ListCategories(cmd.Context())
			if err != nil {
				return err
			}
			wallets, err := client.ListWallets(cmd.Context())
			if err != nil {
				return err
			}
			categoryNames := make(map[string]string, len(categories.Categories))
			for _, item := range categories.Categories {
				categoryNames[item.ID] = item.Name
			}
			walletNames := make(map[string]string, len(wallets.Wallets))
			for _, item := range wallets.Wallets {
				walletNames[item.ID] = item.Name
			}

			switch strings.ToLower(strings.TrimSpace(output)) {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(resp.Incomes)
			case "csv":
				w := csv.NewWriter(cmd.OutOrStdout())
				_ = w.Write([]string{"date", "wallet", "category", "source", "amount", "notes"})
				for _, inc := range resp.Incomes {
					notes := ""
					if inc.Notes != nil {
						notes = *inc.Notes
					}
					category := ""
					if inc.CategoryID != nil {
						category = *inc.CategoryID
						if name := categoryNames[category]; name != "" {
							category = name
						}
					}
					wallet := walletNames[inc.WalletID]
					if wallet == "" {
						wallet = inc.WalletID
					}
					_ = w.Write([]string{inc.IncomeDate, wallet, category, inc.SourceName, fmt.Sprintf("%s %.2f", inc.Currency, inc.Amount), notes})
				}
				w.Flush()
				if err := w.Error(); err != nil {
					return err
				}
				if resp.NextPagination != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Next pagination: %s\n", resp.NextPagination)
				}
				return nil
			default:
				renderIncomeTable(cmd.OutOrStdout(), resp.Incomes, categoryNames, walletNames)
				if resp.NextPagination != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Next pagination: %s\n", resp.NextPagination)
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&output, "output", "table", "Output format: table, json, csv")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum rows")
	cmd.Flags().StringVar(&pagination, "pagination", "", "Pagination cursor")
	cmd.Flags().StringVar(&from, "from", "", "From date YYYY-MM-DD")
	cmd.Flags().StringVar(&to, "to", "", "To date YYYY-MM-DD")
	cmd.Flags().StringVarP(&categoryID, "category-id", "c", "", "Category ID")

	return cmd
}

func newIncomeAddCmd() *cobra.Command {
	var (
		amount       float64
		walletID     string
		walletName   string
		categoryID   string
		categoryName string
		dateValue    string
		source       string
		notes        string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new income",
		RunE: func(cmd *cobra.Command, args []string) error {
			if amount <= 0 {
				return fmt.Errorf("amount must be greater than zero")
			}
			if strings.TrimSpace(walletID) == "" && strings.TrimSpace(walletName) == "" {
				return fmt.Errorf("wallet or wallet-id is required")
			}

			client := newAPIClient()
			resolvedWalletID, _, err := resolveWalletReference(cmd.Context(), client, walletID, walletName)
			if err != nil {
				return err
			}
			var resolvedCategoryID *string
			if strings.TrimSpace(categoryID) != "" || strings.TrimSpace(categoryName) != "" {
				id, _, err := resolveCategoryReference(cmd.Context(), client, categoryID, categoryName)
				if err != nil {
					return err
				}
				resolvedCategoryID = &id
			}

			parsedDate, err := parseDateArg(dateValue)
			if err != nil {
				return err
			}

			req := CreateIncomeRequest{
				WalletID:   resolvedWalletID,
				CategoryID: resolvedCategoryID,
				SourceName: strings.TrimSpace(source),
				Amount:     amount,
				Currency:   strings.ToUpper(strings.TrimSpace(viper.GetString("base_currency"))),
				IncomeDate: parsedDate,
			}
			if req.Currency == "" {
				req.Currency = "USD"
			}
			if strings.TrimSpace(notes) != "" {
				req.Notes = &notes
			}

			resp, err := client.CreateIncome(cmd.Context(), req)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created income %s\n", resp.ID)
			return nil
		},
	}

	cmd.Flags().Float64VarP(&amount, "amount", "a", 0, "Income amount")
	cmd.Flags().StringVarP(&walletName, "wallet", "w", "", "Wallet name")
	cmd.Flags().StringVar(&walletID, "wallet-id", "", "Wallet ID")
	cmd.Flags().StringVarP(&categoryName, "category", "c", "", "Category name")
	cmd.Flags().StringVar(&categoryID, "category-id", "", "Category ID")
	cmd.Flags().StringVarP(&dateValue, "date", "d", "today", "Date (YYYY-MM-DD, today, yesterday)")
	cmd.Flags().StringVarP(&source, "source", "s", "", "Source name")
	cmd.Flags().StringVarP(&notes, "notes", "n", "", "Notes")
	_ = cmd.MarkFlagRequired("amount")
	_ = cmd.MarkFlagRequired("wallet")

	return cmd
}

func newIncomeDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <income-id>",
		Short: "Delete an income",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			return client.DeleteIncome(cmd.Context(), args[0])
		},
	}
}

func renderIncomeTable(out io.Writer, incomes []Income, categoryNames, walletNames map[string]string) {
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Date", "Wallet", "Category", "Source", "Amount", "Notes"})

	for _, inc := range incomes {
		notes := ""
		if inc.Notes != nil {
			notes = *inc.Notes
		}
		category := ""
		if inc.CategoryID != nil {
			category = *inc.CategoryID
			if name := categoryNames[category]; name != "" {
				category = name
			}
		}
		wallet := walletNames[inc.WalletID]
		if wallet == "" {
			wallet = inc.WalletID
		}
		table.Append([]string{inc.IncomeDate, wallet, category, inc.SourceName, fmt.Sprintf("%s %.2f", inc.Currency, inc.Amount), notes})
	}

	table.Render()
}

func newExpenseCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "expense", Short: "Expense commands"}
	cmd.AddCommand(newExpenseAddCmd())
	cmd.AddCommand(newExpenseListCmd())
	cmd.AddCommand(newExpenseDeleteCmd())
	return cmd
}

func newExpenseAddCmd() *cobra.Command {
	var (
		amount       float64
		walletID     string
		walletName   string
		categoryID   string
		categoryName string
		dateValue    string
		currency     string
		merchant     string
		notes        string
		fxRateToBase float64
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new expense",
		RunE: func(cmd *cobra.Command, args []string) error {
			if amount <= 0 {
				return fmt.Errorf("amount must be greater than zero")
			}
			if strings.TrimSpace(walletID) == "" && strings.TrimSpace(walletName) == "" {
				return fmt.Errorf("wallet or wallet-id is required")
			}
			if strings.TrimSpace(categoryID) == "" && strings.TrimSpace(categoryName) == "" {
				return fmt.Errorf("category or category-id is required")
			}

			client := newAPIClient()

			resolvedWalletID, resolvedWalletName, err := resolveWalletReference(cmd.Context(), client, walletID, walletName)
			if err != nil {
				return err
			}
			resolvedCategoryID, resolvedCategoryName, err := resolveCategoryReference(cmd.Context(), client, categoryID, categoryName)
			if err != nil {
				return err
			}

			if currency == "" {
				currency = strings.ToUpper(strings.TrimSpace(viper.GetString("base_currency")))
				if currency == "" {
					currency = "USD"
				}
			}

			parsedDate, err := parseDateArg(dateValue)
			if err != nil {
				return err
			}

			var merchantPtr *string
			if strings.TrimSpace(merchant) != "" {
				value := strings.TrimSpace(merchant)
				merchantPtr = &value
			}
			var notesPtr *string
			if strings.TrimSpace(notes) != "" {
				value := strings.TrimSpace(notes)
				notesPtr = &value
			}

			resp, err := client.CreateExpense(cmd.Context(), CreateExpenseRequest{
				WalletID:     resolvedWalletID,
				Amount:       amount,
				Currency:     currency,
				FXRateToBase: fxRateToBase,
				CategoryID:   resolvedCategoryID,
				Merchant:     merchantPtr,
				Date:         parsedDate,
				Notes:        notesPtr,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Added expense %s for %s in %s\n", resp.ID, resolvedCategoryName, resolvedWalletName)
			return nil
		},
	}

	cmd.Flags().Float64VarP(&amount, "amount", "a", 0, "Expense amount")
	cmd.Flags().StringVarP(&walletName, "wallet", "w", "", "Wallet name")
	cmd.Flags().StringVar(&walletID, "wallet-id", "", "Wallet ID")
	cmd.Flags().StringVarP(&categoryName, "category", "c", "", "Category name")
	cmd.Flags().StringVar(&categoryID, "category-id", "", "Category ID")
	cmd.Flags().StringVarP(&dateValue, "date", "d", "today", "Date (YYYY-MM-DD, today, yesterday)")
	cmd.Flags().StringVarP(&currency, "currency", "C", "", "Currency code")
	cmd.Flags().StringVarP(&merchant, "merchant", "m", "", "Merchant")
	cmd.Flags().StringVarP(&notes, "notes", "n", "", "Notes")
	cmd.Flags().Float64Var(&fxRateToBase, "fx-rate-to-base", 1, "FX rate to base currency")

	return cmd
}

func newExpenseListCmd() *cobra.Command {
	var (
		limit      int
		pagination string
		from       string
		to         string
		categoryID string
		category   string
	)
	var output string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List expenses",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			resolvedCategoryID := strings.TrimSpace(categoryID)
			if resolvedCategoryID == "" && strings.TrimSpace(category) != "" {
				value, _, err := resolveCategoryReference(cmd.Context(), client, "", category)
				if err != nil {
					return err
				}
				resolvedCategoryID = value
			}

			resp, err := client.ListExpenses(cmd.Context(), limit, pagination, from, to, resolvedCategoryID)
			if err != nil {
				return err
			}

			categories, err := client.ListCategories(cmd.Context())
			if err != nil {
				return err
			}
			wallets, err := client.ListWallets(cmd.Context())
			if err != nil {
				return err
			}
			categoryNames := make(map[string]string, len(categories.Categories))
			for _, item := range categories.Categories {
				categoryNames[item.ID] = item.Name
			}
			walletNames := make(map[string]string, len(wallets.Wallets))
			for _, item := range wallets.Wallets {
				walletNames[item.ID] = item.Name
			}

			switch strings.ToLower(strings.TrimSpace(output)) {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(resp.Expenses)
			case "csv":
				w := csv.NewWriter(cmd.OutOrStdout())
				_ = w.Write([]string{"date", "wallet", "category", "merchant", "amount", "notes"})
				for _, expense := range resp.Expenses {
					merchant := ""
					if expense.Merchant != nil {
						merchant = *expense.Merchant
					}
					notes := ""
					if expense.Notes != nil {
						notes = *expense.Notes
					}
					categoryName := categoryNames[expense.CategoryID]
					if categoryName == "" {
						categoryName = expense.CategoryID
					}
					walletName := walletNames[expense.WalletID]
					if walletName == "" {
						walletName = expense.WalletID
					}
					_ = w.Write([]string{expense.Date, walletName, categoryName, merchant, fmt.Sprintf("%s %.2f", expense.Currency, expense.Amount), notes})
				}
				w.Flush()
				if err := w.Error(); err != nil {
					return err
				}
				if resp.NextPagination != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Next pagination: %s\n", resp.NextPagination)
				}
				return nil
			default:
				renderExpenseTable(cmd.OutOrStdout(), resp.Expenses, categoryNames, walletNames)
				if resp.NextPagination != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Next pagination: %s\n", resp.NextPagination)
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&output, "output", "table", "Output format: table, json, csv")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum rows")
	cmd.Flags().StringVar(&pagination, "pagination", "", "Pagination cursor")
	cmd.Flags().StringVar(&from, "from", "", "From date YYYY-MM-DD")
	cmd.Flags().StringVar(&to, "to", "", "To date YYYY-MM-DD")
	cmd.Flags().StringVarP(&category, "category", "c", "", "Category name")
	cmd.Flags().StringVar(&categoryID, "category-id", "", "Category ID")

	return cmd

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum rows")
	cmd.Flags().StringVar(&pagination, "pagination", "", "Pagination cursor")
	cmd.Flags().StringVar(&from, "from", "", "From date YYYY-MM-DD")
	cmd.Flags().StringVar(&to, "to", "", "To date YYYY-MM-DD")
	cmd.Flags().StringVarP(&category, "category", "c", "", "Category name")
	cmd.Flags().StringVar(&categoryID, "category-id", "", "Category ID")

	return cmd
}

func newExpenseDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <expense-id>",
		Short: "Delete an expense",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			return client.DeleteExpense(cmd.Context(), args[0])
		},
	}
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Configuration commands"}
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigViewCmd())
	return cmd
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			viper.Set(args[0], args[1])
			return saveConfig()
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), viper.GetString(args[0]))
			return nil
		},
	}
}

func newConfigViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "View all configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			keys := make([]string, 0, len(viper.AllKeys()))
			for _, key := range viper.AllKeys() {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				value := viper.GetString(key)
				if key == "access_token" || key == "refresh_token" {
					value = "[redacted]"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", key, value)
			}
			return nil
		},
	}
}

func parseDateArg(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "today" {
		return time.Now().Format("2006-01-02"), nil
	}
	if trimmed == "yesterday" {
		return time.Now().AddDate(0, 0, -1).Format("2006-01-02"), nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid date: %s", value)
	}
	return parsed.Format("2006-01-02"), nil
}

func resolveCategoryReference(ctx context.Context, client *APIClient, idValue, nameValue string) (string, string, error) {
	if strings.TrimSpace(idValue) != "" {
		return strings.TrimSpace(idValue), strings.TrimSpace(nameValue), nil
	}
	resp, err := client.ListCategories(ctx)
	if err != nil {
		return "", "", err
	}
	match := findByName(resp.Categories, nameValue)
	if match == nil {
		return "", "", fmt.Errorf("category not found: %s", nameValue)
	}
	return match.ID, match.Name, nil
}

func resolveWalletReference(ctx context.Context, client *APIClient, idValue, nameValue string) (string, string, error) {
	if strings.TrimSpace(idValue) != "" {
		return strings.TrimSpace(idValue), strings.TrimSpace(nameValue), nil
	}
	resp, err := client.ListWallets(ctx)
	if err != nil {
		return "", "", err
	}
	match := findWalletByName(resp.Wallets, nameValue)
	if match == nil {
		return "", "", fmt.Errorf("wallet not found: %s", nameValue)
	}
	return match.ID, match.Name, nil
}

func findByName(items []Category, name string) *Category {
	trimmed := strings.TrimSpace(name)
	for i := range items {
		if strings.EqualFold(items[i].Name, trimmed) {
			return &items[i]
		}
	}
	return nil
}

func findWalletByName(items []Wallet, name string) *Wallet {
	trimmed := strings.TrimSpace(name)
	for i := range items {
		if strings.EqualFold(items[i].Name, trimmed) {
			return &items[i]
		}
	}
	return nil
}

func renderExpenseTable(out io.Writer, expenses []Expense, categoryNames, walletNames map[string]string) {
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Date", "Wallet", "Category", "Merchant", "Amount", "Notes"})

	for _, expense := range expenses {
		merchant := ""
		if expense.Merchant != nil {
			merchant = *expense.Merchant
		}
		notes := ""
		if expense.Notes != nil {
			notes = *expense.Notes
		}
		categoryName := categoryNames[expense.CategoryID]
		if categoryName == "" {
			categoryName = expense.CategoryID
		}
		walletName := walletNames[expense.WalletID]
		if walletName == "" {
			walletName = expense.WalletID
		}
		table.Append([]string{expense.Date, walletName, categoryName, merchant, fmt.Sprintf("%s %.2f", expense.Currency, expense.Amount), notes})
	}

	table.Render()
}
