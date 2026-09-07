package championgrpc

import (
	"context"
	"fmt"
	"time"

	championv1 "lol-timer/gen/champion/v1"
	"lol-timer/internal/services"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const DefaultRequestTimeout = 2 * time.Second

type Client struct {
	connection *grpc.ClientConn
	client     championv1.ChampionServiceClient
	timeout    time.Duration
}

func NewClient(
	address string,
	timeout time.Duration,
) (*Client, error) {
	return newClient(
		address,
		timeout,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
		grpc.WithChainUnaryInterceptor(
			UnaryClientLoggingInterceptor(),
		),
	)
}

func newClient(
	address string,
	timeout time.Duration,
	options ...grpc.DialOption,
) (*Client, error) {
	if address == "" {
		return nil, fmt.Errorf(
			"champion gRPC address is required",
		)
	}

	if timeout <= 0 {
		return nil, fmt.Errorf(
			"champion gRPC timeout must be positive",
		)
	}

	connection, err := grpc.NewClient(
		address,
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create champion gRPC connection: %w",
			err,
		)
	}

	return &Client{
		connection: connection,
		client: championv1.NewChampionServiceClient(
			connection,
		),
		timeout: timeout,
	}, nil
}

func (c *Client) Get(
	ctx context.Context,
	championID int,
) (services.ChampionInfo, error) {
	if championID <= 0 {
		return services.ChampionInfo{}, status.Error(
			codes.InvalidArgument,
			"champion ID must be positive",
		)
	}

	requestCtx, cancel := context.WithTimeout(
		ctx,
		c.timeout,
	)
	defer cancel()

	response, err := c.client.GetChampion(
		requestCtx,
		&championv1.GetChampionRequest{
			ChampionId: int32(championID),
		},
	)
	if err != nil {
		return services.ChampionInfo{}, fmt.Errorf(
			"get champion %d: %w",
			championID,
			err,
		)
	}

	champion := response.GetChampion()
	if champion == nil {
		return services.ChampionInfo{}, status.Error(
			codes.Internal,
			"champion response is empty",
		)
	}

	return services.ChampionInfo{
		ID:       int(champion.GetId()),
		Name:     champion.GetName(),
		ImageURL: champion.GetImageUrl(),
	}, nil
}

func (c *Client) List(
	ctx context.Context,
) (
	[]services.ChampionInfo,
	string,
	error,
) {
	requestCtx, cancel := context.WithTimeout(
		ctx,
		c.timeout,
	)
	defer cancel()

	response, err := c.client.ListChampions(
		requestCtx,
		&championv1.ListChampionsRequest{},
	)
	if err != nil {
		return nil, "", fmt.Errorf(
			"list champions: %w",
			err,
		)
	}

	protoChampions := response.GetChampions()

	champions := make(
		[]services.ChampionInfo,
		0,
		len(protoChampions),
	)

	for _, champion := range protoChampions {
		if champion == nil {
			continue
		}

		champions = append(
			champions,
			services.ChampionInfo{
				ID: int(
					champion.GetId(),
				),
				Name:     champion.GetName(),
				ImageURL: champion.GetImageUrl(),
			},
		)
	}

	return champions,
		response.GetCatalogVersion(),
		nil
}

func (c *Client) Close() error {
	if c == nil || c.connection == nil {
		return nil
	}

	if err := c.connection.Close(); err != nil {
		return fmt.Errorf(
			"close champion gRPC connection: %w",
			err,
		)
	}

	return nil
}
