package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"spendsense-backend/internal/auth"
	"spendsense-backend/internal/category"
	"spendsense-backend/internal/expense"
	"spendsense-backend/internal/infra"
	"spendsense-backend/internal/middleware"
	"spendsense-backend/internal/wallet"
)

type Server struct {
	db              *infra.Database
	authService     *auth.AuthService
	authMiddleware  *middleware.AuthMiddleware
	mux             *http.ServeMux
	expenseService  *expense.Service
	walletService   *wallet.Service
	categoryService *category.Service
}

func NewServer(databaseURL, jwtSecret string) (*Server, error) {
	db, err := infra.NewDatabase(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	server := &Server{
		db:              db,
		authService:     auth.NewAuthService(db, auth.NewJWTManager(jwtSecret)),
		authMiddleware:  middleware.NewAuthMiddleware(auth.NewJWTManager(jwtSecret)),
		mux:             http.NewServeMux(),
		expenseService:  expense.NewService(expense.NewRepository(db)),
		walletService:   wallet.NewService(wallet.NewRepository(db)),
		categoryService: category.NewService(category.NewRepository(db)),
	}

	server.routes()
	return server, nil
}

func (s *Server) Start(port string) error {
	defer s.db.Close()

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      s.mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}
