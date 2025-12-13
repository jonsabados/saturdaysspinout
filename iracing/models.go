package iracing

const (
	EventTypePractice  = 2
	EventTypeQualify   = 3
	EventTypeTimeTrial = 4
	EventTypeRace      = 5
)

// Track contains track information from race results
type Track struct {
	TrackID    int    `json:"track_id"`
	TrackName  string `json:"track_name"`
	ConfigName string `json:"config_name"`
}

// SessionResult represents a single race/practice/qualifying result from the search API
type SessionResult struct {
	SessionID    int64  `json:"session_id"`
	SubsessionID int64  `json:"subsession_id"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`

	// License info
	LicenseCategoryID int    `json:"license_category_id"`
	LicenseCategory   string `json:"license_category"`

	// Event info
	EventType     int    `json:"event_type"`
	EventTypeName string `json:"event_type_name"`
	NumDrivers    int    `json:"num_drivers"`

	// Race stats (may be -1 for non-race sessions)
	NumCautions       int `json:"num_cautions"`
	NumCautionLaps    int `json:"num_caution_laps"`
	NumLeadChanges    int `json:"num_lead_changes"`
	EventAverageLap   int `json:"event_average_lap"`
	EventBestLapTime  int `json:"event_best_lap_time"`
	EventLapsComplete int `json:"event_laps_complete"`

	// Winner info
	WinnerGroupID int64  `json:"winner_group_id"`
	WinnerName    string `json:"winner_name"`
	WinnerAI      bool   `json:"winner_ai"`
	DriverChanges bool   `json:"driver_changes"`

	// Driver-specific results (the cust_id we searched for)
	CustID                  int64 `json:"cust_id"`
	StartingPosition        int   `json:"starting_position"`
	FinishPosition          int   `json:"finish_position"`
	StartingPositionInClass int   `json:"starting_position_in_class"`
	FinishPositionInClass   int   `json:"finish_position_in_class"`
	LapsComplete            int   `json:"laps_complete"`
	LapsLed                 int   `json:"laps_led"`
	Incidents               int   `json:"incidents"`

	// Car info
	CarID             int    `json:"car_id"`
	CarName           string `json:"car_name"`
	CarNameAbbrev     string `json:"car_name_abbreviated"`
	CarClassID        int    `json:"car_class_id"`
	CarClassName      string `json:"car_class_name"`
	CarClassShortName string `json:"car_class_short_name"`

	// Track info
	Track Track `json:"track"`

	// Series/Season info
	OfficialSession        bool   `json:"official_session"`
	SeriesID               int    `json:"series_id"`
	SeriesName             string `json:"series_name"`
	SeriesShortName        string `json:"series_short_name"`
	SeasonID               int    `json:"season_id"`
	SeasonYear             int    `json:"season_year"`
	SeasonQuarter          int    `json:"season_quarter"`
	SeasonLicenseGroup     int    `json:"season_license_group"`
	SeasonLicenseGroupName string `json:"season_license_group_name"`
	RaceWeekNum            int    `json:"race_week_num"`

	// Points (may be -1 for non-scoring sessions)
	EventStrengthOfField int  `json:"event_strength_of_field"`
	ChampPoints          int  `json:"champ_points"`
	DropRace             bool `json:"drop_race"`
}
