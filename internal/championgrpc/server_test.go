package championgrpc

import (
	"context"
	"testing"
	"time"

	championv1 "lol-timer/gen/champion/v1"
	"lol-timer/internal/services"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type championCatalogStub struct {
	find func(
		championID int,
	) (services.ChampionInfo, bool)

	list func() []services.ChampionInfo

	version func() string
}

func (s *championCatalogStub) Find(
	championID int,
) (services.ChampionInfo, bool) {
	return s.find(championID)
}

func (s *championCatalogStub) List() []services.ChampionInfo {
	if s.list == nil {
		return nil
	}

	return s.list()
}

func (s *championCatalogStub) Version() string {
	if s.version == nil {
		return ""
	}

	return s.version()
}

func TestServerGetChampion(
	t *testing.T,
) {
	catalog := &championCatalogStub{
		find: func(
			championID int,
		) (services.ChampionInfo, bool) {
			if championID != 103 {
				t.Fatalf(
					"expected champion ID 103, got %d",
					championID,
				)
			}

			return services.ChampionInfo{
				ID:       103,
				Name:     "Ahri",
				ImageURL: "https://example.com/Ahri.png",
			}, true
		},

		version: func() string {
			return "16.18.1"
		},
	}

	server := NewServer(catalog)

	response, err := server.GetChampion(
		context.Background(),
		&championv1.GetChampionRequest{
			ChampionId: 103,
		},
	)
	if err != nil {
		t.Fatalf("get champion: %v", err)
	}

	champion := response.GetChampion()
	if champion == nil {
		t.Fatal("expected champion response")
	}

	if champion.GetId() != 103 {
		t.Errorf(
			"expected champion ID 103, got %d",
			champion.GetId(),
		)
	}

	if champion.GetName() != "Ahri" {
		t.Errorf(
			"expected name %q, got %q",
			"Ahri",
			champion.GetName(),
		)
	}

	if champion.GetImageUrl() !=
		"https://example.com/Ahri.png" {
		t.Errorf(
			"unexpected image URL %q",
			champion.GetImageUrl(),
		)
	}

	if response.GetCatalogVersion() != "16.18.1" {
		t.Errorf(
			"expected catalog version %q, got %q",
			"16.18.1",
			response.GetCatalogVersion(),
		)
	}
}

func TestServerGetChampionStatusCodes(
	t *testing.T,
) {
	catalog := &championCatalogStub{
		find: func(
			_ int,
		) (services.ChampionInfo, bool) {
			return services.ChampionInfo{}, false
		},
	}

	server := NewServer(catalog)

	tests := []struct {
		name     string
		ctx      func() context.Context
		request  *championv1.GetChampionRequest
		expected codes.Code
	}{
		{
			name: "nil request",
			ctx: func() context.Context {
				return context.Background()
			},
			request:  nil,
			expected: codes.InvalidArgument,
		},
		{
			name: "invalid champion ID",
			ctx: func() context.Context {
				return context.Background()
			},
			request: &championv1.GetChampionRequest{
				ChampionId: 0,
			},
			expected: codes.InvalidArgument,
		},
		{
			name: "champion not found",
			ctx: func() context.Context {
				return context.Background()
			},
			request: &championv1.GetChampionRequest{
				ChampionId: 999,
			},
			expected: codes.NotFound,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(
					context.Background(),
				)
				cancel()

				return ctx
			},
			request: &championv1.GetChampionRequest{
				ChampionId: 103,
			},
			expected: codes.Canceled,
		},
		{
			name: "expired deadline",
			ctx: func() context.Context {
				ctx, cancel := context.WithDeadline(
					context.Background(),
					time.Now().Add(-time.Second),
				)

				t.Cleanup(cancel)

				return ctx
			},
			request: &championv1.GetChampionRequest{
				ChampionId: 103,
			},
			expected: codes.DeadlineExceeded,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				_, err := server.GetChampion(
					test.ctx(),
					test.request,
				)

				if actual := status.Code(err); actual != test.expected {
					t.Errorf(
						"expected status %s, got %s: %v",
						test.expected,
						actual,
						err,
					)
				}
			},
		)
	}
}

func TestServerListChampions(
	t *testing.T,
) {
	catalog := &championCatalogStub{
		find: func(
			_ int,
		) (services.ChampionInfo, bool) {
			return services.ChampionInfo{}, false
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

	server := NewServer(catalog)

	response, err := server.ListChampions(
		context.Background(),
		&championv1.ListChampionsRequest{},
	)
	if err != nil {
		t.Fatalf("list champions: %v", err)
	}

	champions := response.GetChampions()

	if len(champions) != 2 {
		t.Fatalf(
			"expected two champions, got %d",
			len(champions),
		)
	}

	if champions[0].GetId() != 103 {
		t.Errorf(
			"expected first champion ID 103, got %d",
			champions[0].GetId(),
		)
	}

	if champions[1].GetId() != 222 {
		t.Errorf(
			"expected second champion ID 222, got %d",
			champions[1].GetId(),
		)
	}

	if response.GetCatalogVersion() != "16.18.1" {
		t.Errorf(
			"expected catalog version %q, got %q",
			"16.18.1",
			response.GetCatalogVersion(),
		)
	}
}

func TestServerListChampionsRejectsNilRequest(
	t *testing.T,
) {
	server := NewServer(
		&championCatalogStub{
			find: func(
				_ int,
			) (services.ChampionInfo, bool) {
				return services.ChampionInfo{}, false
			},
		},
	)

	_, err := server.ListChampions(
		context.Background(),
		nil,
	)

	if actual := status.Code(err); actual != codes.InvalidArgument {
		t.Errorf(
			"expected status %s, got %s: %v",
			codes.InvalidArgument,
			actual,
			err,
		)
	}
}
