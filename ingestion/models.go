package ingestion

type RaceIngestionRequest struct {
	DriverID           int64  `json:"driverID"`
	IRacingAccessToken string `json:"iRacingAccessToken"`
	NotifyConnectionID string `json:"notifyConnectionID"`
}
