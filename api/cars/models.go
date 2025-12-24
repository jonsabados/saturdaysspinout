package cars

import "github.com/jonsabados/saturdaysspinout/cars"

type Car struct {
	ID                   int64    `json:"id"`
	Name                 string   `json:"name"`
	NameAbbreviated      string   `json:"nameAbbreviated"`
	Make                 string   `json:"make"`
	Model                string   `json:"model"`
	Description          string   `json:"description"`
	Weight               int      `json:"weight"`
	HPUnderHood          int      `json:"hpUnderHood"`
	HPActual             int      `json:"hpActual"`
	Categories           []string `json:"categories"`
	LogoURL              string   `json:"logoUrl"`
	SmallImageURL        string   `json:"smallImageUrl"`
	LargeImageURL        string   `json:"largeImageUrl"`
	HasHeadlights        bool     `json:"hasHeadlights"`
	HasMultipleDryTires  bool     `json:"hasMultipleDryTires"`
	RainEnabled          bool     `json:"rainEnabled"`
	FreeWithSubscription bool     `json:"freeWithSubscription"`
	Retired              bool     `json:"retired"`
}

func carFromDomain(c cars.Car) Car {
	return Car{
		ID:                   c.ID,
		Name:                 c.Name,
		NameAbbreviated:      c.NameAbbreviated,
		Make:                 c.Make,
		Model:                c.Model,
		Description:          c.Description,
		Weight:               c.Weight,
		HPUnderHood:          c.HPUnderHood,
		HPActual:             c.HPActual,
		Categories:           c.Categories,
		LogoURL:              c.LogoURL,
		SmallImageURL:        c.SmallImageURL,
		LargeImageURL:        c.LargeImageURL,
		HasHeadlights:        c.HasHeadlights,
		HasMultipleDryTires:  c.HasMultipleDryTires,
		RainEnabled:          c.RainEnabled,
		FreeWithSubscription: c.FreeWithSubscription,
		Retired:              c.Retired,
	}
}