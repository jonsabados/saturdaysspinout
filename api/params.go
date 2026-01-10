package api

// begin path parameters
const DriverIDPathParam = "driver_id"

// begin url parameters
const (
	StartTimeQueryParam       = "startTime"
	EndTimeQueryParam         = "endTime"
	PageQueryParam            = "page"
	ResultsPerPageParam       = "resultsPerPage"
	DefaultResultsPerPage int = 10

	// Analytics query params
	GroupByQueryParam     = "groupBy"
	GranularityQueryParam = "granularity"
	SeriesIDQueryParam    = "seriesId"
	CarIDQueryParam       = "carId"
	TrackIDQueryParam     = "trackId"
)
