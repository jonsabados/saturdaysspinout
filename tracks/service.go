package tracks

import (
	"context"

	"github.com/jonsabados/saturdaysspinout/iracing"
)

type IRacingClient interface {
	GetTracks(ctx context.Context, accessToken string) ([]iracing.TrackInfo, error)
	GetTrackAssets(ctx context.Context, accessToken string) (map[int64]iracing.TrackAssets, error)
}

type TrackMapLayers struct {
	Background  string
	Inactive    string
	Active      string
	Pitroad     string
	StartFinish string
	Turns       string
}

type Track struct {
	ID                int64
	Name              string
	ConfigName        string
	Category          string
	Location          string
	CornersPerLap     int
	LengthMiles       float64
	Description       string
	LogoURL           string
	SmallImageURL     string
	LargeImageURL     string
	TrackMapURL       string
	TrackMapLayers    TrackMapLayers
	IsDirt            bool
	IsOval            bool
	HasNightLighting  bool
	RainEnabled       bool
	FreeWithSub       bool
	Retired           bool
	PitRoadSpeedLimit int
}

type Service struct {
	client IRacingClient
}

func NewService(client IRacingClient) *Service {
	return &Service{client: client}
}

func (s *Service) GetAll(ctx context.Context, accessToken string) ([]Track, error) {
	trackInfos, err := s.client.GetTracks(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	assets, err := s.client.GetTrackAssets(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	tracks := make([]Track, 0, len(trackInfos))
	for _, info := range trackInfos {
		track := Track{
			ID:                info.TrackID,
			Name:              info.TrackName,
			ConfigName:        info.ConfigName,
			Category:          info.Category,
			Location:          info.Location,
			CornersPerLap:     info.CornersPerLap,
			LengthMiles:       info.TrackConfigLength,
			IsDirt:            info.IsDirt,
			IsOval:            info.IsOval,
			HasNightLighting:  info.NightLighting,
			RainEnabled:       info.RainEnabled,
			FreeWithSub:       info.FreeWithSubscription,
			Retired:           info.Retired,
			PitRoadSpeedLimit: info.PitRoadSpeedLimit,
		}

		// Merge asset data if available
		if asset, ok := assets[info.TrackID]; ok {
			track.Description = asset.DetailCopy
			track.TrackMapURL = asset.TrackMap
			track.TrackMapLayers = TrackMapLayers{
				Background:  asset.TrackMapLayers.Background,
				Inactive:    asset.TrackMapLayers.Inactive,
				Active:      asset.TrackMapLayers.Active,
				Pitroad:     asset.TrackMapLayers.Pitroad,
				StartFinish: asset.TrackMapLayers.StartFinish,
				Turns:       asset.TrackMapLayers.Turns,
			}

			if asset.Logo != "" {
				track.LogoURL = iracing.ImageBaseURL + asset.Logo
			}
			if asset.SmallImage != "" && asset.Folder != "" {
				track.SmallImageURL = iracing.ImageBaseURL + asset.Folder + "/" + asset.SmallImage
			}
			if asset.LargeImage != "" && asset.Folder != "" {
				track.LargeImageURL = iracing.ImageBaseURL + asset.Folder + "/" + asset.LargeImage
			}
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}
