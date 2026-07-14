package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IceWhaleTech/CasaOS/pkg/config"
	"github.com/IceWhaleTech/CasaOS/pkg/jwt"
	"github.com/IceWhaleTech/CasaOS/pkg/middleware"
	"github.com/IceWhaleTech/CasaOS/route"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func main() {
	// Inicjalizacja konfiguracji (w tym JWT secret)
	if err := config.InitConfig(); err != nil {
		log.Fatalf("Failed to init config: %v", err)
	}

	// Inicjalizacja loggera
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)

	// Inicjalizacja aplikacji
	e := echo.New()
	
	// Middleware
	e.Use(middleware.Logger)
	e.Use(middleware.Recover)
	
	// Publiczne endpointy (bez autoryzacji)
	e.POST("/api/v1/auth/login", route.Login)
	e.POST("/api/v1/auth/register", route.Register)
	e.GET("/api/v1/health", route.HealthCheck)
	
	// Grupa z autoryzacją
	api := e.Group("/api/v1")
	api.Use(middleware.AuthMiddleware)
	{
		api.GET("/user/profile", route.GetProfile)
		api.POST("/user/logout", route.Logout)
		api.GET("/devices", route.GetDevices)
		api.POST("/devices", route.AddDevice)
		
		// Endpointy admina
		admin := api.Group("/admin")
		admin.Use(middleware.AdminMiddleware)
		{
			admin.GET("/users", route.GetUsers)
			admin.POST("/system/restart", route.RestartSystem)
			admin.POST("/system/update", route.UpdateSystem)
		}
	}

	// Obsługa sygnałów dla graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		logrus.Info("Shutting down gracefully...")
		
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
		defer shutdownCancel()
		
		if err := e.Shutdown(shutdownCtx); err != nil {
			logrus.Errorf("Error during shutdown: %v", err)
		}
		cancel()
	}()

	// Uruchom serwer
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	logrus.Infof("Starting CasaOS on port %s", port)
	if err := e.Start(":" + port); err != http.ErrServerClosed {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
