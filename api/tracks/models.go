package tracks

import "github.com/jonsabados/saturdaysspinout/tracks"

type Track struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	ConfigName        string  `json:"configName"`
	Category          string  `json:"category"`
	Location          string  `json:"location"`
	CornersPerLap     int     `json:"cornersPerLap"`
	LengthMiles       float64 `json:"lengthMiles"`
	Description       string  `json:"description"`
	LogoURL           string  `json:"logoUrl"`
	SmallImageURL     string  `json:"smallImageUrl"`
	LargeImageURL     string  `json:"largeImageUrl"`
	TrackMapURL       string  `json:"trackMapUrl"`
	IsDirt            bool    `json:"isDirt"`
	IsOval            bool    `json:"isOval"`
	HasNightLighting  bool    `json:"hasNightLighting"`
	RainEnabled       bool    `json:"rainEnabled"`
	FreeWithSub       bool    `json:"freeWithSubscription"`
	Retired           bool    `json:"retired"`
	PitRoadSpeedLimit int     `json:"pitRoadSpeedLimit"`
}

func trackFromDomain(t tracks.Track) Track {
	return Track{
		ID:                t.ID,
		Name:              t.Name,
		ConfigName:        t.ConfigName,
		Category:          t.Category,
		Location:          t.Location,
		CornersPerLap:     t.CornersPerLap,
		LengthMiles:       t.LengthMiles,
		Description:       t.Description,
		LogoURL:           t.LogoURL,
		SmallImageURL:     t.SmallImageURL,
		LargeImageURL:     t.LargeImageURL,
		TrackMapURL:       t.TrackMapURL,
		IsDirt:            t.IsDirt,
		IsOval:            t.IsOval,
		HasNightLighting:  t.HasNightLighting,
		RainEnabled:       t.RainEnabled,
		FreeWithSub:       t.FreeWithSub,
		Retired:           t.Retired,
		PitRoadSpeedLimit: t.PitRoadSpeedLimit,
	}
}