package cars

import (
	"context"

	"github.com/jonsabados/saturdaysspinout/iracing"
)

type IRacingClient interface {
	GetCars(ctx context.Context, accessToken string) ([]iracing.CarInfo, error)
	GetCarAssets(ctx context.Context, accessToken string) (map[int64]iracing.CarAssets, error)
}

type Car struct {
	ID                   int64
	Name                 string
	NameAbbreviated      string
	Make                 string
	Model                string
	Description          string
	Weight               int
	HPUnderHood          int
	HPActual             int
	Categories           []string
	LogoURL              string
	SmallImageURL        string
	LargeImageURL        string
	HasHeadlights        bool
	HasMultipleDryTires  bool
	RainEnabled          bool
	FreeWithSubscription bool
	Retired              bool
}

type Service struct {
	client IRacingClient
}

func NewService(client IRacingClient) *Service {
	return &Service{client: client}
}

func (s *Service) GetAll(ctx context.Context, accessToken string) ([]Car, error) {
	carInfos, err := s.client.GetCars(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	assets, err := s.client.GetCarAssets(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	cars := make([]Car, 0, len(carInfos))
	for _, info := range carInfos {
		car := Car{
			ID:                   info.CarID,
			Name:                 info.CarName,
			NameAbbreviated:      info.CarNameAbbreviated,
			Make:                 info.CarMake,
			Model:                info.CarModel,
			Weight:               info.CarWeight,
			HPUnderHood:          info.HPUnderHood,
			HPActual:             info.HPActual,
			Categories:           info.Categories,
			HasHeadlights:        info.HasHeadlights,
			HasMultipleDryTires:  info.HasMultipleDryTires,
			RainEnabled:          info.RainEnabled,
			FreeWithSubscription: info.FreeWithSubscription,
			Retired:              info.Retired,
		}

		// Merge asset data if available
		if asset, ok := assets[info.CarID]; ok {
			car.Description = asset.DetailCopy

			if asset.Logo != "" {
				car.LogoURL = iracing.ImageBaseURL + asset.Logo
			}
			if asset.SmallImage != "" && asset.Folder != "" {
				car.SmallImageURL = iracing.ImageBaseURL + asset.Folder + "/" + asset.SmallImage
			}
			if asset.LargeImage != "" && asset.Folder != "" {
				car.LargeImageURL = iracing.ImageBaseURL + asset.Folder + "/" + asset.LargeImage
			}
		}

		cars = append(cars, car)
	}

	return cars, nil
}