package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fekuna/omnipos-customer-service/config"
	"github.com/fekuna/omnipos-customer-service/internal/customer/handler"
	"github.com/fekuna/omnipos-customer-service/internal/customer/listener"
	"github.com/fekuna/omnipos-customer-service/internal/customer/repository"
	"github.com/fekuna/omnipos-customer-service/internal/customer/usecase"
	"github.com/fekuna/omnipos-customer-service/internal/middleware"
	"github.com/fekuna/omnipos-pkg/broker"
	"github.com/fekuna/omnipos-pkg/database/postgres"
	"github.com/fekuna/omnipos-pkg/logger"
	customerv1 "github.com/fekuna/omnipos-proto/gen/go/omnipos/customer/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. Load Configuration
	cfg := config.LoadEnv()

	// 2. Initialize Logger
	logConfig := &logger.ZapLoggerConfig{
		IsDevelopment:     false,
		Encoding:          "json",
		Level:             "info",
		DisableCaller:     false,
		DisableStacktrace: false,
	}

	if cfg.Server.AppEnv == "development" {
		logConfig.IsDevelopment = true
		logConfig.Encoding = "console"
		logConfig.Level = "debug"
	}

	appLogger := logger.NewZapLogger(logConfig)
	defer appLogger.Sync()

	// 3. Connect to Database
	db, err := postgres.NewPostgres(&postgres.Config{
		Host:            cfg.Postgres.Host,
		Port:            cfg.Postgres.Port,
		User:            cfg.Postgres.User,
		Password:        cfg.Postgres.Password,
		DBName:          cfg.Postgres.DBName,
		SSLMode:         cfg.Postgres.SSLMode,
		MaxOpenConns:    cfg.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Postgres.ConnMaxIdleTime,
	})
	if err != nil {
		appLogger.Fatal("Could not connect to database", zap.Error(err))
	}
	defer db.Close()
	appLogger.Info("Connected to PostgreSQL database", zap.String("db_name", cfg.Postgres.DBName))

	// 4. Initialize Components
	repo := repository.NewPGRepository(db)
	uc := usecase.NewCustomerUseCase(repo, appLogger)
	h := handler.NewCustomerHandler(uc, appLogger)

	// Initialize Middleware
	authInterceptor := middleware.NewAuthContextInterceptor(appLogger)

	// 4.5 Initialize Kafka Listener
	kafkaConsumer := broker.NewConsumer(&broker.Config{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
		GroupID: cfg.Kafka.GroupID,
	})
	appLogger.Info("Connected to Kafka Consumer", zap.Strings("brokers", cfg.Kafka.Brokers), zap.String("topic", cfg.Kafka.Topic))

	importCmdContext := context.Background() // Need context for listener? Listener.Start takes context.
	// But Start() usually blocks?
	// listener implementation: func (l *CustomerListener) Start(ctx context.Context)
	// We run it in goroutine.

	customerListener := listener.NewCustomerListener(kafkaConsumer, uc, appLogger)
	go customerListener.Start(importCmdContext)

	// 5. Start gRPC Server
	port := cfg.Server.GRPCPort
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.Unary()),
	)

	// Register Services
	customerv1.RegisterCustomerServiceServer(grpcServer, h)

	// Register Reflection
	reflection.Register(grpcServer)

	appLogger.Info("Starting Customer Service gRPC server", zap.String("port", port))

	// Graceful Shutdown
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			appLogger.Fatal("failed to serve", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")
	grpcServer.GracefulStop()
	appLogger.Info("Server stopped")
}
