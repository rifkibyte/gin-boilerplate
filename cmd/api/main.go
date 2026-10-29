package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/rifkibyte/gin-boilerplate/internal/config"
	"github.com/rifkibyte/gin-boilerplate/internal/auth"
	authadapter "github.com/rifkibyte/gin-boilerplate/internal/auth/adapter"
	authhttp "github.com/rifkibyte/gin-boilerplate/internal/auth/http"
	"github.com/rifkibyte/gin-boilerplate/internal/middleware"
	"github.com/rifkibyte/gin-boilerplate/internal/router"
	"github.com/rifkibyte/gin-boilerplate/internal/user"
	useradapter "github.com/rifkibyte/gin-boilerplate/internal/user/adapter"
	userhttp "github.com/rifkibyte/gin-boilerplate/internal/user/http"
	"github.com/rifkibyte/gin-boilerplate/pkg/cache"
	"github.com/rifkibyte/gin-boilerplate/pkg/database"
	"github.com/rifkibyte/gin-boilerplate/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(logger.Config{
		Level:   cfg.Log.Level,
		Format:  logger.Format(cfg.Log.Format),
		Service: cfg.Log.Service,
	})

	defer func() {
		_ = log.Sync()
	}()

	// Graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Database.
	db, err := database.NewPostgres(cfg.DB)
	if err != nil {
		log.Fatal("database connection failed", zap.Error(err))
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Error("database close error", zap.Error(err))
		}
	}()

	// Redis.
	redisClient := cache.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("redis close error", zap.Error(err))
		}
	}()

	// Wire user feature.
	userRepo := useradapter.NewUserRepositoryGORM(db)
	userSvc := user.NewService(userRepo, userRepo, log)
	userHandler := userhttp.NewHandler(userSvc)

	// Wire auth feature.
	jwtIssuer := authadapter.NewJWTIssuer(cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	tokenStore := authadapter.NewRedisTokenStore(redisClient)
	mailer := authadapter.NewSMTPMailer(cfg.Mail)
	authSvc := auth.NewService(userRepo, userRepo, jwtIssuer, tokenStore, mailer, log)
	authHandler := authhttp.NewHandler(authSvc)

	authMiddleware := middleware.Auth(func(token string) (string, string, error) {
		return jwtIssuer.VerifyAccessToken(context.Background(), token)
	})
	adminMiddleware := middleware.RequireRole(string(user.RoleAdmin))

	// Router.
	r := router.New(userHandler)
	r.POST("/api/v1/auth/register", authHandler.Register)
	r.POST("/api/v1/auth/login", authHandler.Login)
	r.POST("/api/v1/auth/refresh", authHandler.Refresh)
	r.POST("/api/v1/auth/logout", authHandler.Logout)
	r.GET("/api/v1/users", authMiddleware, adminMiddleware, userHandler.List)

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      r,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		log.Info("server starting", zap.String("addr", cfg.HTTP.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	}

	log.Info("server stopped")
}

var _ = gin.New
var _ = time.Second
