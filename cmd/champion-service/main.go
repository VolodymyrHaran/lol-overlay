package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	championv1 "lol-timer/gen/champion/v1"
	"lol-timer/internal/championgrpc"
	"lol-timer/internal/logger"
	"lol-timer/internal/services"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	defaultGRPCAddress = ":50051"

	catalogLoadTimeout = 15 * time.Second
	shutdownTimeout    = 10 * time.Second
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(
			".env file not found, using system environment variables",
		)
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	slog.SetDefault(
		logger.New(logLevel),
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	grpcAddress := os.Getenv(
		"CHAMPION_GRPC_ADDRESS",
	)
	if grpcAddress == "" {
		grpcAddress = defaultGRPCAddress
	}

	catalog := services.NewChampionService()

	loadCtx, loadCancel := context.WithTimeout(
		ctx,
		catalogLoadTimeout,
	)

	if err := catalog.Load(loadCtx); err != nil {
		loadCancel()
		log.Fatal(
			"load champion catalog: ",
			err,
		)
	}

	loadCancel()

	listener, err := net.Listen(
		"tcp",
		grpcAddress,
	)
	if err != nil {
		log.Fatal(
			"listen for gRPC: ",
			err,
		)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			championgrpc.UnaryServerLoggingInterceptor(),
		),
	)

	championServer :=
		championgrpc.NewServer(catalog)

	championv1.RegisterChampionServiceServer(
		grpcServer,
		championServer,
	)

	healthServer := health.NewServer()

	healthv1.RegisterHealthServer(
		grpcServer,
		healthServer,
	)

	healthServer.SetServingStatus(
		"",
		healthv1.HealthCheckResponse_SERVING,
	)

	healthServer.SetServingStatus(
		championv1.ChampionService_ServiceDesc.
			ServiceName,
		healthv1.HealthCheckResponse_SERVING,
	)

	serverError := make(chan error, 1)

	go func() {
		log.Printf(
			"champion gRPC service listening on %s",
			grpcAddress,
		)

		serverError <- grpcServer.Serve(
			listener,
		)
	}()

	select {
	case err := <-serverError:
		if err != nil &&
			!errors.Is(
				err,
				grpc.ErrServerStopped,
			) {
			log.Fatal(
				"serve champion gRPC service: ",
				err,
			)
		}

		return

	case <-ctx.Done():
		log.Println(
			"champion gRPC service shutdown requested",
		)
	}

	healthServer.SetServingStatus(
		"",
		healthv1.HealthCheckResponse_NOT_SERVING,
	)

	healthServer.SetServingStatus(
		championv1.ChampionService_ServiceDesc.
			ServiceName,
		healthv1.HealthCheckResponse_NOT_SERVING,
	)

	gracefulStopCompleted := make(
		chan struct{},
	)

	go func() {
		grpcServer.GracefulStop()
		close(gracefulStopCompleted)
	}()

	shutdownTimer := time.NewTimer(
		shutdownTimeout,
	)
	defer shutdownTimer.Stop()

	select {
	case <-gracefulStopCompleted:
		log.Println(
			"champion gRPC service stopped",
		)

	case <-shutdownTimer.C:
		log.Println(
			"champion gRPC graceful shutdown timed out",
		)

		grpcServer.Stop()
	}
}
