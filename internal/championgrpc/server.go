package championgrpc

import (
	"context"

	championv1 "lol-timer/gen/champion/v1"
	"lol-timer/internal/services"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChampionCatalog interface {
	Find(
		championID int,
	) (services.ChampionInfo, bool)

	List() []services.ChampionInfo

	Version() string
}

type Server struct {
	championv1.UnimplementedChampionServiceServer

	catalog ChampionCatalog
}

func NewServer(
	catalog ChampionCatalog,
) *Server {
	return &Server{
		catalog: catalog,
	}
}

func (s *Server) GetChampion(
	ctx context.Context,
	request *championv1.GetChampionRequest,
) (*championv1.GetChampionResponse, error) {
	if err := contextStatusError(ctx); err != nil {
		return nil, err
	}

	if request == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required",
		)
	}

	if request.GetChampionId() <= 0 {
		return nil, status.Error(
			codes.InvalidArgument,
			"champion ID must be positive",
		)
	}

	champion, exists := s.catalog.Find(
		int(request.GetChampionId()),
	)
	if !exists {
		return nil, status.Errorf(
			codes.NotFound,
			"champion %d not found",
			request.GetChampionId(),
		)
	}

	return &championv1.GetChampionResponse{
		Champion:       toProtoChampion(champion),
		CatalogVersion: s.catalog.Version(),
	}, nil
}

func (s *Server) ListChampions(
	ctx context.Context,
	request *championv1.ListChampionsRequest,
) (*championv1.ListChampionsResponse, error) {
	if err := contextStatusError(ctx); err != nil {
		return nil, err
	}

	if request == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required",
		)
	}

	champions := s.catalog.List()

	protoChampions := make(
		[]*championv1.Champion,
		0,
		len(champions),
	)

	for _, champion := range champions {
		protoChampions = append(
			protoChampions,
			toProtoChampion(champion),
		)
	}

	return &championv1.ListChampionsResponse{
		Champions:      protoChampions,
		CatalogVersion: s.catalog.Version(),
	}, nil
}

func toProtoChampion(
	champion services.ChampionInfo,
) *championv1.Champion {
	return &championv1.Champion{
		Id:       int32(champion.ID),
		Name:     champion.Name,
		ImageUrl: champion.ImageURL,
	}
}

func contextStatusError(
	ctx context.Context,
) error {
	if err := ctx.Err(); err != nil {
		return status.FromContextError(err).Err()
	}

	return nil
}
