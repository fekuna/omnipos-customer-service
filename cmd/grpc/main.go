package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/fekuna/omnipos-customer-service/config"
	"github.com/fekuna/omnipos-customer-service/internal/customer/handler"
	"github.com/fekuna/omnipos-customer-service/internal/customer/listener"
	"github.com/fekuna/omnipos-customer-service/internal/customer/repository"
	"github.com/fekuna/omnipos-customer-service/internal/customer/usecase"
	"github.com/fekuna/omnipos-pkg/broker"
	"github.com/fekuna/omnipos-pkg/database/postgres"
	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/fekuna/omnipos-pkg/middleware"
	customerv1 "github.com/fekuna/omnipos-proto/proto/customer/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. Load Config
	cfg := config.LoadEnv()

	// 2. Initialize Logger
	logConfig := &logger.ZapLoggerConfig{
		IsDevelopment:     cfg.Server.AppEnv == "dev",
		DisableCaller:     cfg.Logger.DisableCaller,
		DisableStacktrace: cfg.Logger.DisableStacktrace,
		Level:             cfg.Logger.Level,
		Encoding:          cfg.Logger.Encoding,
	}
	log := logger.NewZapLogger(logConfig)
	// NewZapLogger returns interface, if it returned error handled here?
	// pkg/logger/zap.go NewZapLogger does NOT return error.
	// But main.go line 38 checked for error!
	// Step 3382: func NewZapLogger(cfg *ZapLoggerConfig) ZapLogger
	// It does NOT return error.
	// So remove error check.
	defer log.Sync()

	log.Info("Starting OmniPOS Customer Service", zap.String("env", cfg.Server.AppEnv))

	// ... (i18n init)

	// 4. Initialize Database
	pgConfig := &postgres.Config{
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
	}
	db, err := postgres.NewPostgres(pgConfig)
	if err != nil {
		log.Fatal("Could not connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Could not ping database", zap.Error(err))
	}
	log.Info("Connected to database")

	// 5. Initialize Layers
	repo := repository.NewPGRepository(db)
	useCase := usecase.NewCustomerUseCase(repo, log)
	customerHandler := handler.NewCustomerHandler(useCase, log)

	// 5.1 Initialize Kafka
	kafkaConsumer := broker.NewConsumer(&broker.Config{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
		GroupID: cfg.Kafka.GroupID,
	})
	log.Info("Connected to Kafka Consumer", zap.Strings("brokers", cfg.Kafka.Brokers), zap.String("topic", cfg.Kafka.Topic))
	customerListener := listener.NewCustomerListener(kafkaConsumer, useCase, log)
	go customerListener.Start(context.Background())

	// 6. Start gRPC Server
	lis, err := net.Listen("tcp", cfg.Server.GRPCPort)
	if err != nil {
		log.Fatal("Failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.ContextInterceptor()), // Enable i18n/Timezone propagation
	)
	customerv1.RegisterCustomerServiceServer(grpcServer, customerHandler)

	// Enable reflection for debugging (e.g. grpcurl)
	reflection.Register(grpcServer)

	// Graceful Shutdown
	go func() {
		log.Info(fmt.Sprintf("gRPC server listening on %s", cfg.Server.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	grpcServer.GracefulStop()

	// Close DB connection (defer handles it, but good to be explicit/log)
	// db.Close() comes from defer.

	log.Info("Server exited")
}
