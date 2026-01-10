package series

import (
	"context"

	"github.com/jonsabados/saturdaysspinout/iracing"
)

type IRacingClient interface {
	GetSeries(ctx context.Context, accessToken string) ([]iracing.SeriesInfo, error)
}

type Series struct {
	ID        int
	Name      string
	ShortName string
	Category  string
	LogoURL   string
	Active    bool
	Official  bool
}

type Service struct {
	client IRacingClient
}

func NewService(client IRacingClient) *Service {
	return &Service{client: client}
}

func (s *Service) GetAll(ctx context.Context, accessToken string) ([]Series, error) {
	seriesInfos, err := s.client.GetSeries(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	result := make([]Series, 0, len(seriesInfos))
	for _, info := range seriesInfos {
		series := Series{
			ID:        info.SeriesID,
			Name:      info.SeriesName,
			ShortName: info.SeriesShortName,
			Category:  info.Category,
			Active:    info.Active,
			Official:  info.Official,
		}

		if info.Logo != "" {
			series.LogoURL = iracing.ImageBaseURL + info.Logo
		}

		result = append(result, series)
	}

	return result, nil
}