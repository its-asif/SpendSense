package httpapi

import (
	"net/http"
	"strings"
	"time"

	"spendsense-backend/internal/middleware"
	"spendsense-backend/internal/wallet"

	"github.com/google/uuid"
)

type walletRequest struct {
	Name           string  `json:"name"`
	WalletType     string  `json:"wallet_type"`
	Provider       *string `json:"provider,omitempty"`
	AccountNumber  *string `json:"account_number,omitempty"`
	AccountName    *string `json:"account_name,omitempty"`
	Currency       string  `json:"currency"`
	OpeningBalance float64 `json:"opening_balance,omitempty"`
}

type walletResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	WalletType     string  `json:"wallet_type"`
	Provider       *string `json:"provider,omitempty"`
	AccountNumber  *string `json:"account_number,omitempty"`
	AccountName    *string `json:"account_name,omitempty"`
	Currency       string  `json:"currency"`
	OpeningBalance float64 `json:"opening_balance"`
	CurrentBalance float64 `json:"current_balance"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func (s *Server) registerWalletRoutes() {
	s.mux.Handle("/api/v1/wallets", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleCreateListWallets)))
	s.mux.Handle("/api/v1/wallets/", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleWalletByID)))
}

func (s *Server) handleCreateListWallets(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req walletRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}
		created, err := s.walletService.CreateWallet(r.Context(), userID, wallet.CreateRequest{
			Name: req.Name, WalletType: req.WalletType, Provider: req.Provider, AccountNumber: req.AccountNumber, AccountName: req.AccountName, Currency: req.Currency, OpeningBalance: req.OpeningBalance,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toWalletResponse(created))
		return
	case http.MethodGet:
		list, err := s.walletService.ListWallets(r.Context(), userID)
		if err != nil {
			writeError(w, err)
			return
		}
		resp := make([]walletResponse, 0, len(list))
		for _, it := range list {
			resp = append(resp, toWalletResponse(it))
		}
		writeJSON(w, http.StatusOK, map[string]any{"wallets": resp})
		return
	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
}

func (s *Server) handleWalletByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/wallets/")
	idStr = strings.Trim(idStr, "/")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_ID", "Invalid wallet id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		wObj, err := s.walletService.GetWallet(r.Context(), userID, id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toWalletResponse(wObj))
		return
	case http.MethodPut:
		var req walletRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}
		updated, err := s.walletService.UpdateWallet(r.Context(), userID, id, wallet.UpdateRequest{
			Name: req.Name, WalletType: req.WalletType, Provider: req.Provider, AccountNumber: req.AccountNumber, AccountName: req.AccountName, Currency: req.Currency, IsActive: true, CurrentBalance: req.OpeningBalance,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toWalletResponse(updated))
		return
	case http.MethodDelete:
		if err := s.walletService.DeleteWallet(r.Context(), userID, id); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusNoContent, nil)
		return
	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
}

func toWalletResponse(wObj *wallet.Wallet) walletResponse {
	return walletResponse{
		ID: wObj.ID.String(), Name: wObj.Name, WalletType: wObj.WalletType, Provider: wObj.Provider, AccountNumber: wObj.AccountNumber, AccountName: wObj.AccountName,
		Currency: wObj.Currency, OpeningBalance: wObj.OpeningBalance, CurrentBalance: wObj.CurrentBalance, IsActive: wObj.IsActive,
		CreatedAt: wObj.CreatedAt.Format(time.RFC3339), UpdatedAt: wObj.UpdatedAt.Format(time.RFC3339),
	}
}
