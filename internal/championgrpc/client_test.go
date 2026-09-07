package championgrpc

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	championv1 "lol-timer/gen/champion/v1"
	"lol-timer/internal/services"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const testBufferSize = 1024 * 1024

func TestClientGetAndListChampions(
	t *testing.T,
) {
	var findCalls atomic.Int32

	catalog := &championCatalogStub{
		find: func(
			championID int,
		) (services.ChampionInfo, bool) {
			findCalls.Add(1)

			switch championID {
			case 103:
				return services.ChampionInfo{
					ID:       103,
					Name:     "Ahri",
					ImageURL: "https://example.com/Ahri.png",
				}, true

			case 222:
				return services.ChampionInfo{
					ID:       222,
					Name:     "Jinx",
					ImageURL: "https://example.com/Jinx.png",
				}, true

			default:
				return services.ChampionInfo{}, false
			}
		},

		list: func() []services.ChampionInfo {
			return []services.ChampionInfo{
				{
					ID:       103,
					Name:     "Ahri",
					ImageURL: "https://example.com/Ahri.png",
				},
				{
					ID:       222,
					Name:     "Jinx",
					ImageURL: "https://example.com/Jinx.png",
				},
			}
		},

		version: func() string {
			return "16.18.1"
		},
	}

	client, dialCalls := newBufferClient(
		t,
		NewServer(catalog),
		time.Second,
	)
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf(
				"close champion client: %v",
				err,
			)
		}
	})

	first, err := client.Get(
		context.Background(),
		103,
	)
	if err != nil {
		t.Fatalf(
			"get first champion: %v",
			err,
		)
	}

	if first.Name != "Ahri" {
		t.Errorf(
			"expected first champion %q, got %q",
			"Ahri",
			first.Name,
		)
	}

	second, err := client.Get(
		context.Background(),
		222,
	)
	if err != nil {
		t.Fatalf(
			"get second champion: %v",
			err,
		)
	}

	if second.Name != "Jinx" {
		t.Errorf(
			"expected second champion %q, got %q",
			"Jinx",
			second.Name,
		)
	}

	champions, version, err := client.List(
		context.Background(),
	)
	if err != nil {
		t.Fatalf("list champions: %v", err)
	}

	if len(champions) != 2 {
		t.Fatalf(
			"expected two champions, got %d",
			len(champions),
		)
	}

	if version != "16.18.1" {
		t.Errorf(
			"expected version %q, got %q",
			"16.18.1",
			version,
		)
	}

	if actual := findCalls.Load(); actual != 2 {
		t.Errorf(
			"expected two Find calls, got %d",
			actual,
		)
	}

	if actual := dialCalls.Load(); actual != 1 {
		t.Errorf(
			"expected one reused connection, got %d dials",
			actual,
		)
	}
}

func TestClientPreservesNotFoundStatus(
	t *testing.T,
) {
	catalog := &championCatalogStub{
		find: func(
			_ int,
		) (services.ChampionInfo, bool) {
			return services.ChampionInfo{}, false
		},
	}

	client, _ := newBufferClient(
		t,
		NewServer(catalog),
		time.Second,
	)
	t.Cleanup(func() {
		_ = client.Close()
	})

	_, err := client.Get(
		context.Background(),
		999,
	)

	if actual := status.Code(err); actual != codes.NotFound {
		t.Errorf(
			"expected status %s, got %s: %v",
			codes.NotFound,
			actual,
			err,
		)
	}
}

func TestClientRejectsInvalidChampionID(
	t *testing.T,
) {
	catalog := &championCatalogStub{
		find: func(
			_ int,
		) (services.ChampionInfo, bool) {
			return services.ChampionInfo{}, false
		},
	}

	client, dialCalls := newBufferClient(
		t,
		NewServer(catalog),
		time.Second,
	)
	t.Cleanup(func() {
		_ = client.Close()
	})

	_, err := client.Get(
		context.Background(),
		0,
	)

	if actual := status.Code(err); actual != codes.InvalidArgument {
		t.Errorf(
			"expected status %s, got %s: %v",
			codes.InvalidArgument,
			actual,
			err,
		)
	}

	if actual := dialCalls.Load(); actual != 0 {
		t.Errorf(
			"expected no network dial, got %d",
			actual,
		)
	}
}

func TestClientAppliesRequestTimeout(
	t *testing.T,
) {
	client, _ := newBufferClient(
		t,
		&blockingChampionServer{},
		20*time.Millisecond,
	)
	t.Cleanup(func() {
		_ = client.Close()
	})

	startedAt := time.Now()

	_, err := client.Get(
		context.Background(),
		103,
	)

	if actual := status.Code(err); actual != codes.DeadlineExceeded {
		t.Errorf(
			"expected status %s, got %s: %v",
			codes.DeadlineExceeded,
			actual,
			err,
		)
	}

	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Errorf(
			"expected timeout near client deadline, took %s",
			elapsed,
		)
	}
}

type blockingChampionServer struct {
	championv1.UnimplementedChampionServiceServer
}

func (s *blockingChampionServer) GetChampion(
	ctx context.Context,
	_ *championv1.GetChampionRequest,
) (*championv1.GetChampionResponse, error) {
	<-ctx.Done()

	return nil,
		status.FromContextError(
			ctx.Err(),
		).Err()
}

func newBufferClient(
	t *testing.T,
	server championv1.ChampionServiceServer,
	timeout time.Duration,
) (*Client, *atomic.Int32) {
	t.Helper()

	listener := bufconn.Listen(
		testBufferSize,
	)

	grpcServer := grpc.NewServer()

	championv1.RegisterChampionServiceServer(
		grpcServer,
		server,
	)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()

		if err := listener.Close(); err != nil {
			t.Errorf(
				"close buffer listener: %v",
				err,
			)
		}
	})

	var dialCalls atomic.Int32

	client, err := newClient(
		"passthrough:///champion-test",
		timeout,
		grpc.WithContextDialer(
			func(
				_ context.Context,
				_ string,
			) (net.Conn, error) {
				dialCalls.Add(1)
				return listener.Dial()
			},
		),
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		t.Fatalf(
			"create buffer client: %v",
			err,
		)
	}

	return client, &dialCalls
}
