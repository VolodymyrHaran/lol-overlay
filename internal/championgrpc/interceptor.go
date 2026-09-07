package championgrpc

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		startedAt := time.Now()

		response, err := handler(
			ctx,
			request,
		)

		slog.Info(
			"gRPC server request",
			"method", info.FullMethod,
			"code", status.Code(err).String(),
			"duration", time.Since(startedAt),
		)

		return response, err
	}
}

func UnaryClientLoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		request any,
		response any,
		connection *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		options ...grpc.CallOption,
	) error {
		startedAt := time.Now()

		err := invoker(
			ctx,
			method,
			request,
			response,
			connection,
			options...,
		)

		slog.Info(
			"gRPC client request",
			"method", method,
			"code", status.Code(err).String(),
			"duration", time.Since(startedAt),
		)

		return err
	}
}
