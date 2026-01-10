package series

import "github.com/jonsabados/saturdaysspinout/series"

type Series struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	Category  string `json:"category"`
	LogoURL   string `json:"logoUrl"`
	Active    bool   `json:"active"`
	Official  bool   `json:"official"`
}

func seriesFromDomain(s series.Series) Series {
	return Series{
		ID:        s.ID,
		Name:      s.Name,
		ShortName: s.ShortName,
		Category:  s.Category,
		LogoURL:   s.LogoURL,
		Active:    s.Active,
		Official:  s.Official,
	}
}