package httpapi

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"spendsense-backend/internal/auth"
	"spendsense-backend/internal/category"
	"spendsense-backend/internal/expense"
	"spendsense-backend/internal/income"
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
	incomeService   *income.Service
	walletService   *wallet.Service
	categoryService *category.Service
	httpServer      *http.Server
	cleanupCancel   context.CancelFunc
	shutdownOnce    sync.Once
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
		incomeService:   income.NewService(income.NewRepository(db)),
		walletService:   wallet.NewService(wallet.NewRepository(db)),
		categoryService: category.NewService(category.NewRepository(db)),
	}

	server.routes()
	return server, nil
}

func (s *Server) Start(port string) error {
	cleanupCtx, cancelCleanup := context.WithCancel(context.Background())
	s.cleanupCancel = cancelCleanup
	go s.startRefreshTokenCleanupJob(cleanupCtx, time.Hour)

	s.httpServer = &http.Server{
		Addr: ":" + port,
		Handler: middleware.Recoverer(
			middleware.RequestLogger(
				middleware.NewRateLimiter(180, time.Minute).Middleware(s.mux),
			),
		),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		_ = s.Shutdown(context.Background())
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	var shutdownErr error
	s.shutdownOnce.Do(func() {
		if s.cleanupCancel != nil {
			s.cleanupCancel()
		}

		if s.httpServer != nil {
			if err := s.httpServer.Shutdown(ctx); err != nil {
				shutdownErr = err
			}
		}

		s.db.Close()
	})

	return shutdownErr
}

func (s *Server) startRefreshTokenCleanupJob(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deleted, err := s.authService.CleanupExpiredRefreshTokens(ctx)
			if err != nil {
				log.Printf("refresh token cleanup failed: %v", err)
				continue
			}

			if deleted > 0 {
				log.Printf("refresh token cleanup removed %d expired tokens", deleted)
			}
		}
	}
}
