package iracing

import "time"

// EventType represents the type of event in iRacing (practice, qualify, race, etc.)
type EventType int

const (
	EventTypePractice  EventType = 2
	EventTypeQualify   EventType = 3
	EventTypeTimeTrial EventType = 4
	EventTypeRace      EventType = 5
)

// Track contains track information from race results.
// Fields are populated differently depending on the client method used.
type Track struct {
	// TrackID is the unique identifier for the track. Populated by all methods.
	TrackID int64 `json:"track_id"`
	// TrackName is the display name of the track. Populated by all methods.
	TrackName string `json:"track_name"`
	// ConfigName is the track configuration name. Populated by all methods.
	ConfigName string `json:"config_name"`
	// Category is the track category (e.g., "oval", "road"). Only populated by GetSessionResults.
	Category string `json:"category"`
	// CategoryID is the numeric track category identifier. Only populated by GetSessionResults.
	CategoryID int `json:"category_id"`
}

// SeriesResult represents a single race/practice/qualifying result from the search API
type SeriesResult struct {
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

// SessionResult represents the full response from GetSessionResults.
type SessionResult struct {
	SubsessionID            int64              `json:"subsession_id"`
	AllowedLicenses         []AllowedLicense   `json:"allowed_licenses"`
	AssociatedSubsessionIDs []int64            `json:"associated_subsession_ids"`
	CanProtest              bool               `json:"can_protest"`
	CarClasses              []CarClass         `json:"car_classes"`
	CautionType             int                `json:"caution_type"`
	CooldownMinutes         int                `json:"cooldown_minutes"`
	CornersPerLap           int                `json:"corners_per_lap"`
	DamageModel             int                `json:"damage_model"`
	DriverChangeParam1      int                `json:"driver_change_param1"`
	DriverChangeParam2      int                `json:"driver_change_param2"`
	DriverChangeRule        int                `json:"driver_change_rule"`
	DriverChanges           bool               `json:"driver_changes"`
	EndTime                 time.Time          `json:"end_time"`
	EventAverageLap         int                `json:"event_average_lap"`
	EventBestLapTime        int                `json:"event_best_lap_time"`
	EventLapsComplete       int                `json:"event_laps_complete"`
	EventStrengthOfField    int                `json:"event_strength_of_field"`
	EventType               int                `json:"event_type"`
	EventTypeName           string             `json:"event_type_name"`
	HeatInfoID              int                `json:"heat_info_id"`
	LicenseCategory         string             `json:"license_category"`
	LicenseCategoryID       int                `json:"license_category_id"`
	LimitMinutes            int                `json:"limit_minutes"`
	MaxTeamDrivers          int                `json:"max_team_drivers"`
	MaxWeeks                int                `json:"max_weeks"`
	MinTeamDrivers          int                `json:"min_team_drivers"`
	NumCautionLaps          int                `json:"num_caution_laps"`
	NumCautions             int                `json:"num_cautions"`
	NumDrivers              int                `json:"num_drivers"`
	NumLapsForQualAverage   int                `json:"num_laps_for_qual_average"`
	NumLapsForSoloAverage   int                `json:"num_laps_for_solo_average"`
	NumLeadChanges          int                `json:"num_lead_changes"`
	OfficialSession         bool               `json:"official_session"`
	PointsType              string             `json:"points_type"`
	PrivateSessionID        int                `json:"private_session_id"`
	RaceSummary             RaceSummary        `json:"race_summary"`
	RaceWeekNum             int                `json:"race_week_num"`
	ResultsRestricted       bool               `json:"results_restricted"`
	SeasonID                int                `json:"season_id"`
	SeasonName              string             `json:"season_name"`
	SeasonQuarter           int                `json:"season_quarter"`
	SeasonShortName         string             `json:"season_short_name"`
	SeasonYear              int                `json:"season_year"`
	SeriesID                int                `json:"series_id"`
	SeriesLogo              string             `json:"series_logo"`
	SeriesName              string             `json:"series_name"`
	SeriesShortName         string             `json:"series_short_name"`
	SessionID               int64              `json:"session_id"`
	SessionResults          []SimSessionResult `json:"session_results"`
	SessionSplits           []SessionSplit     `json:"session_splits"`
	SpecialEventType        int                `json:"special_event_type"`
	StartTime               time.Time          `json:"start_time"`
	Track                   Track              `json:"track"`
	TrackState              TrackState         `json:"track_state"`
	Weather                 Weather            `json:"weather"`
}

type AllowedLicense struct {
	GroupName       string `json:"group_name"`
	LicenseGroup    int    `json:"license_group"`
	MaxLicenseLevel int    `json:"max_license_level"`
	MinLicenseLevel int    `json:"min_license_level"`
	ParentID        int    `json:"parent_id"`
}

type CarClass struct {
	CarClassID      int          `json:"car_class_id"`
	ShortName       string       `json:"short_name"`
	Name            string       `json:"name"`
	StrengthOfField int          `json:"strength_of_field"`
	NumEntries      int          `json:"num_entries"`
	CarsInClass     []CarInClass `json:"cars_in_class"`
}

type CarInClass struct {
	CarID int `json:"car_id"`
}

type RaceSummary struct {
	SubsessionID         int64  `json:"subsession_id"`
	AverageLap           int    `json:"average_lap"`
	LapsComplete         int    `json:"laps_complete"`
	NumCautions          int    `json:"num_cautions"`
	NumCautionLaps       int    `json:"num_caution_laps"`
	NumLeadChanges       int    `json:"num_lead_changes"`
	FieldStrength        int    `json:"field_strength"`
	NumOptLaps           int    `json:"num_opt_laps"`
	HasOptPath           bool   `json:"has_opt_path"`
	SpecialEventType     int    `json:"special_event_type"`
	SpecialEventTypeText string `json:"special_event_type_text"`
}

type SimSessionResult struct {
	SimsessionNumber   int            `json:"simsession_number"`
	SimsessionName     string         `json:"simsession_name"`
	SimsessionType     int            `json:"simsession_type"`
	SimsessionTypeName string         `json:"simsession_type_name"`
	SimsessionSubtype  int            `json:"simsession_subtype"`
	WeatherResult      WeatherResult  `json:"weather_result"`
	Results            []DriverResult `json:"results"`
}

type WeatherResult struct {
	AvgSkies                 int     `json:"avg_skies"`
	AvgCloudCoverPct         float64 `json:"avg_cloud_cover_pct"`
	MinCloudCoverPct         float64 `json:"min_cloud_cover_pct"`
	MaxCloudCoverPct         float64 `json:"max_cloud_cover_pct"`
	TempUnits                int     `json:"temp_units"`
	AvgTemp                  float64 `json:"avg_temp"`
	MinTemp                  float64 `json:"min_temp"`
	MaxTemp                  float64 `json:"max_temp"`
	AvgRelHumidity           float64 `json:"avg_rel_humidity"`
	WindUnits                int     `json:"wind_units"`
	AvgWindSpeed             float64 `json:"avg_wind_speed"`
	MinWindSpeed             float64 `json:"min_wind_speed"`
	MaxWindSpeed             float64 `json:"max_wind_speed"`
	AvgWindDir               int     `json:"avg_wind_dir"`
	MaxFog                   float64 `json:"max_fog"`
	FogTimePct               float64 `json:"fog_time_pct"`
	PrecipTimePct            float64 `json:"precip_time_pct"`
	PrecipMM                 float64 `json:"precip_mm"`
	PrecipMM2HrBeforeSession float64 `json:"precip_mm2hr_before_session"`
	SimulatedStartTime       string  `json:"simulated_start_time"`
}

type DriverResult struct {
	CustID                  int64     `json:"cust_id"`
	DisplayName             string    `json:"display_name"`
	AggregateChampPoints    int       `json:"aggregate_champ_points"`
	AI                      bool      `json:"ai"`
	AverageLap              int       `json:"average_lap"`
	BestLapNum              int       `json:"best_lap_num"`
	BestLapTime             int       `json:"best_lap_time"`
	BestNLapsNum            int       `json:"best_nlaps_num"`
	BestNLapsTime           int       `json:"best_nlaps_time"`
	BestQualLapAt           time.Time `json:"best_qual_lap_at"`
	BestQualLapNum          int       `json:"best_qual_lap_num"`
	BestQualLapTime         int       `json:"best_qual_lap_time"`
	CarClassID              int       `json:"car_class_id"`
	CarClassName            string    `json:"car_class_name"`
	CarClassShortName       string    `json:"car_class_short_name"`
	CarID                   int64     `json:"car_id"`
	CarName                 string    `json:"car_name"`
	CarCfg                  int       `json:"carcfg"`
	ChampPoints             int       `json:"champ_points"`
	ClassInterval           int       `json:"class_interval"`
	CountryCode             string    `json:"country_code"`
	Division                int       `json:"division"`
	DivisionName            string    `json:"division_name"`
	DropRace                bool      `json:"drop_race"`
	FinishPosition          int       `json:"finish_position"`
	FinishPositionInClass   int       `json:"finish_position_in_class"`
	FlairID                 int       `json:"flair_id"`
	FlairName               string    `json:"flair_name"`
	FlairShortname          string    `json:"flair_shortname"`
	Friend                  bool      `json:"friend"`
	Helmet                  Helmet    `json:"helmet"`
	Incidents               int       `json:"incidents"`
	Interval                int       `json:"interval"`
	LapsComplete            int       `json:"laps_complete"`
	LapsLead                int       `json:"laps_lead"`
	LeagueAggPoints         int       `json:"league_agg_points"`
	LeaguePoints            int       `json:"league_points"`
	LicenseChangeOval       int       `json:"license_change_oval"`
	LicenseChangeRoad       int       `json:"license_change_road"`
	Livery                  Livery    `json:"livery"`
	MaxPctFuelFill          int       `json:"max_pct_fuel_fill"`
	NewCPI                  float64   `json:"new_cpi"`
	NewLicenseLevel         int       `json:"new_license_level"`
	NewSubLevel             int       `json:"new_sub_level"`
	NewTTRating             int       `json:"new_ttrating"`
	NewIRating              int       `json:"newi_rating"`
	OldCPI                  float64   `json:"old_cpi"`
	OldLicenseLevel         int       `json:"old_license_level"`
	OldSubLevel             int       `json:"old_sub_level"`
	OldTTRating             int       `json:"old_ttrating"`
	OldIRating              int       `json:"oldi_rating"`
	OptLapsComplete         int       `json:"opt_laps_complete"`
	Position                int       `json:"position"`
	QualLapTime             int       `json:"qual_lap_time"`
	ReasonOut               string    `json:"reason_out"`
	ReasonOutID             int       `json:"reason_out_id"`
	StartingPosition        int       `json:"starting_position"`
	StartingPositionInClass int       `json:"starting_position_in_class"`
	Suit                    Suit      `json:"suit"`
	Watched                 bool      `json:"watched"`
	WeightPenaltyKg         int       `json:"weight_penalty_kg"`
}

type Helmet struct {
	Pattern    int    `json:"pattern"`
	Color1     string `json:"color1"`
	Color2     string `json:"color2"`
	Color3     string `json:"color3"`
	FaceType   int    `json:"face_type"`
	HelmetType int    `json:"helmet_type"`
}

type Livery struct {
	CarID        int     `json:"car_id"`
	Pattern      int     `json:"pattern"`
	Color1       string  `json:"color1"`
	Color2       string  `json:"color2"`
	Color3       string  `json:"color3"`
	NumberFont   int     `json:"number_font"`
	NumberColor1 string  `json:"number_color1"`
	NumberColor2 string  `json:"number_color2"`
	NumberColor3 string  `json:"number_color3"`
	NumberSlant  int     `json:"number_slant"`
	Sponsor1     int     `json:"sponsor1"`
	Sponsor2     int     `json:"sponsor2"`
	CarNumber    string  `json:"car_number"`
	WheelColor   *string `json:"wheel_color"`
	RimType      int     `json:"rim_type"`
}

type Suit struct {
	Pattern int    `json:"pattern"`
	Color1  string `json:"color1"`
	Color2  string `json:"color2"`
	Color3  string `json:"color3"`
}

type SessionSplit struct {
	SubsessionID         int64 `json:"subsession_id"`
	EventStrengthOfField int   `json:"event_strength_of_field"`
}

type TrackState struct {
	LeaveMarbles   bool `json:"leave_marbles"`
	PracticeRubber int  `json:"practice_rubber"`
	QualifyRubber  int  `json:"qualify_rubber"`
	RaceRubber     int  `json:"race_rubber"`
	WarmupRubber   int  `json:"warmup_rubber"`
}

type Weather struct {
	AllowFog                      bool    `json:"allow_fog"`
	Fog                           int     `json:"fog"`
	PrecipMM2HrBeforeFinalSession float64 `json:"precip_mm2hr_before_final_session"`
	PrecipMMFinalSession          float64 `json:"precip_mm_final_session"`
	PrecipOption                  int     `json:"precip_option"`
	PrecipTimePct                 float64 `json:"precip_time_pct"`
	RelHumidity                   int     `json:"rel_humidity"`
	SimulatedStartTime            string  `json:"simulated_start_time"`
	Skies                         int     `json:"skies"`
	TempUnits                     int     `json:"temp_units"`
	TempValue                     int     `json:"temp_value"`
	TimeOfDay                     int     `json:"time_of_day"`
	TrackWater                    int     `json:"track_water"`
	Type                          int     `json:"type"`
	Version                       int     `json:"version"`
	WeatherVarInitial             int     `json:"weather_var_initial"`
	WeatherVarOngoing             int     `json:"weather_var_ongoing"`
	WindDir                       int     `json:"wind_dir"`
	WindUnits                     int     `json:"wind_units"`
	WindValue                     int     `json:"wind_value"`
}

// LapDataResponse is the response from GetLapData containing session metadata
// and lap timing information for a specific driver.
type LapDataResponse struct {
	Success         bool               `json:"success"`
	SessionInfo     LapDataSessionInfo `json:"session_info"`
	BestLapNum      int                `json:"best_lap_num"`
	BestLapTime     int                `json:"best_lap_time"`
	BestNLapsNum    int                `json:"best_nlaps_num"`
	BestNLapsTime   int                `json:"best_nlaps_time"`
	BestQualLapNum  int                `json:"best_qual_lap_num"`
	BestQualLapTime int                `json:"best_qual_lap_time"`
	BestQualLapAt   *time.Time         `json:"best_qual_lap_at"`
	LastUpdated     time.Time          `json:"last_updated"`
	GroupID         int64              `json:"group_id"`
	CustID          int64              `json:"cust_id"`
	Name            string             `json:"name"`
	CarID           int                `json:"car_id"`
	LicenseLevel    int                `json:"license_level"`
	Livery          Livery             `json:"livery"`
	Laps            []Lap              `json:"laps,omitempty"`
}

// Lap represents a single lap's timing and event data.
type Lap struct {
	GroupID          int64    `json:"group_id"`
	Name             string   `json:"name"`
	CustID           int64    `json:"cust_id"`
	DisplayName      string   `json:"display_name"`
	LapNumber        int      `json:"lap_number"`
	Flags            int      `json:"flags"`
	Incident         bool     `json:"incident"`
	SessionTime      int      `json:"session_time"`
	SessionStartTime *int     `json:"session_start_time"`
	LapTime          int      `json:"lap_time"`
	TeamFastestLap   bool     `json:"team_fastest_lap"`
	PersonalBestLap  bool     `json:"personal_best_lap"`
	Helmet           Helmet   `json:"helmet"`
	LicenseLevel     int      `json:"license_level"`
	CarNumber        string   `json:"car_number"`
	LapEvents        []string `json:"lap_events"`
	AI               bool     `json:"ai"`
}

// LapDataSessionInfo contains session metadata returned with lap data.
type LapDataSessionInfo struct {
	SubsessionID          int64     `json:"subsession_id"`
	SessionID             int64     `json:"session_id"`
	SimsessionNumber      int       `json:"simsession_number"`
	SimsessionType        int       `json:"simsession_type"`
	SimsessionName        string    `json:"simsession_name"`
	NumLapsForQualAverage int       `json:"num_laps_for_qual_average"`
	NumLapsForSoloAverage int       `json:"num_laps_for_solo_average"`
	EventType             int       `json:"event_type"`
	EventTypeName         string    `json:"event_type_name"`
	PrivateSessionID      int       `json:"private_session_id"`
	SeasonName            string    `json:"season_name"`
	SeasonShortName       string    `json:"season_short_name"`
	SeriesName            string    `json:"series_name"`
	SeriesShortName       string    `json:"series_short_name"`
	StartTime             time.Time `json:"start_time"`
	Track                 Track     `json:"track"`
}

// TrackType represents a track type entry from the track_types array.
type TrackType struct {
	TrackType string `json:"track_type"`
}

// TrackInfo contains full track information from the /data/track/get endpoint.
type TrackInfo struct {
	TrackID                int64       `json:"track_id"`
	TrackName              string      `json:"track_name"`
	ConfigName             string      `json:"config_name"`
	Category               string      `json:"category"`
	CategoryID             int         `json:"category_id"`
	AIEnabled              bool        `json:"ai_enabled"`
	AllowPitlaneCollisions bool        `json:"allow_pitlane_collisions"`
	AllowRollingStart      bool        `json:"allow_rolling_start"`
	AllowStandingStart     bool        `json:"allow_standing_start"`
	AwardExempt            bool        `json:"award_exempt"`
	Closes                 string      `json:"closes"`
	CornersPerLap          int         `json:"corners_per_lap"`
	Created                time.Time   `json:"created"`
	FirstSale              time.Time   `json:"first_sale"`
	FreeWithSubscription   bool        `json:"free_with_subscription"`
	FullyLit               bool        `json:"fully_lit"`
	GridStalls             int         `json:"grid_stalls"`
	HasOptPath             bool        `json:"has_opt_path"`
	HasShortParadeLap      bool        `json:"has_short_parade_lap"`
	HasStartZone           bool        `json:"has_start_zone"`
	HasSvgMap              bool        `json:"has_svg_map"`
	IsDirt                 bool        `json:"is_dirt"`
	IsOval                 bool        `json:"is_oval"`
	IsPsPurchasable        bool        `json:"is_ps_purchasable"`
	LapScoring             int         `json:"lap_scoring"`
	Latitude               float64     `json:"latitude"`
	Location               string      `json:"location"`
	Longitude              float64     `json:"longitude"`
	MaxCars                int         `json:"max_cars"`
	NightLighting          bool        `json:"night_lighting"`
	NumberPitstalls        int         `json:"number_pitstalls"`
	Opens                  string      `json:"opens"`
	PackageID              int         `json:"package_id"`
	PitRoadSpeedLimit      int         `json:"pit_road_speed_limit"`
	Price                  float64     `json:"price"`
	Priority               int         `json:"priority"`
	Purchasable            bool        `json:"purchasable"`
	QualifyLaps            int         `json:"qualify_laps"`
	RainEnabled            bool        `json:"rain_enabled"`
	RestartOnLeft          bool        `json:"restart_on_left"`
	Retired                bool        `json:"retired"`
	SearchFilters          string      `json:"search_filters"`
	SiteURL                string      `json:"site_url"`
	Sku                    int         `json:"sku"`
	SoloLaps               int         `json:"solo_laps"`
	StartOnLeft            bool        `json:"start_on_left"`
	SupportsGripCompound   bool        `json:"supports_grip_compound"`
	TechTrack              bool        `json:"tech_track"`
	TimeZone               string      `json:"time_zone"`
	TrackConfigLength      float64     `json:"track_config_length"`
	TrackDirpath           string      `json:"track_dirpath"`
	TrackTypeID            int         `json:"track_type"`
	TrackTypeText          string      `json:"track_type_text"`
	TrackTypes             []TrackType `json:"track_types"`
	Folder                 string      `json:"folder"`
	Logo                   string      `json:"logo"`
	SmallImage             string      `json:"small_image"`
}

// CarInfo contains full car information from the /data/car/get endpoint.
type CarInfo struct {
	CarID                int64     `json:"car_id"`
	CarName              string    `json:"car_name"`
	CarNameAbbreviated   string    `json:"car_name_abbreviated"`
	CarMake              string    `json:"car_make"`
	CarModel             string    `json:"car_model"`
	CarWeight            int       `json:"car_weight"`
	HPUnderHood          int       `json:"hp_under_hood"`
	HPActual             int       `json:"hp_actual"`
	AiEnabled            bool      `json:"ai_enabled"`
	AllowNumberColors    bool      `json:"allow_number_colors"`
	AllowNumberFont      bool      `json:"allow_number_font"`
	AllowSponsor1        bool      `json:"allow_sponsor1"`
	AllowSponsor2        bool      `json:"allow_sponsor2"`
	AllowWheelColor      bool      `json:"allow_wheel_color"`
	AwardExempt          bool      `json:"award_exempt"`
	CarDirpath           string    `json:"car_dirpath"`
	CarTypes             []CarType `json:"car_types"`
	Categories           []string  `json:"categories"`
	Created              time.Time `json:"created"`
	FirstSale            time.Time `json:"first_sale"`
	ForumURL             string    `json:"forum_url"`
	FreeWithSubscription bool      `json:"free_with_subscription"`
	HasHeadlights        bool      `json:"has_headlights"`
	HasMultipleDryTires  bool      `json:"has_multiple_dry_tire_types"`
	HasRainCapable       bool      `json:"has_rain_capable_tire_types"`
	IsPsPurchasable      bool      `json:"is_ps_purchasable"`
	MaxPowerAdjustPct    float64   `json:"max_power_adjust_pct"`
	MaxWeightPenaltyKg   int       `json:"max_weight_penalty_kg"`
	MinPowerAdjustPct    float64   `json:"min_power_adjust_pct"`
	PackageID            int       `json:"package_id"`
	Patterns             int       `json:"patterns"`
	Price                float64   `json:"price"`
	PriceDisplay         string    `json:"price_display"`
	RainEnabled          bool      `json:"rain_enabled"`
	Retired              bool      `json:"retired"`
	SearchFilters        string    `json:"search_filters"`
	Sku                  int       `json:"sku"`
	CarRules             []CarRule `json:"car_rules"`
	SiteURL              string    `json:"site_url"`
}

// CarType represents a car type entry from the car_types array.
type CarType struct {
	CarType string `json:"car_type"`
}

// CarRule represents a car rule entry.
type CarRule struct {
	RuleCategory string `json:"rule_category"`
	Text         string `json:"text"`
}

// CarAssets contains asset information from the /data/car/assets endpoint.
type CarAssets struct {
	CarID                  int64   `json:"car_id"`
	DetailCopy             string  `json:"detail_copy"`
	DetailScreenshotImages string  `json:"detail_screenshot_images"`
	DetailTechspecsCopy    *string `json:"detail_techspecs_copy"`
	Folder                 string  `json:"folder"`
	GalleryImages          string  `json:"gallery_images"`
	GalleryPrefix          *string `json:"gallery_prefix"`
	LargeImage             string  `json:"large_image"`
	Logo                   string  `json:"logo"`
	SmallImage             string  `json:"small_image"`
	SponsorLogo            *string `json:"sponsor_logo"`
	TemplateRoot           *string `json:"template_root"`
}

// TrackMapLayers contains the SVG layer filenames for a track map.
type TrackMapLayers struct {
	Background  string `json:"background"`
	Inactive    string `json:"inactive"`
	Active      string `json:"active"`
	Pitroad     string `json:"pitroad"`
	StartFinish string `json:"start-finish"`
	Turns       string `json:"turns"`
}

// TrackAssets contains asset information from the /data/track/assets endpoint.
type TrackAssets struct {
	TrackID             int64          `json:"track_id"`
	Coordinates         string         `json:"coordinates"`
	DetailCopy          string         `json:"detail_copy"`
	DetailTechspecsCopy *string        `json:"detail_techspecs_copy"`
	DetailVideo         *string        `json:"detail_video"`
	Folder              string         `json:"folder"`
	GalleryImages       string         `json:"gallery_images"`
	GalleryPrefix       string         `json:"gallery_prefix"`
	LargeImage          string         `json:"large_image"`
	Logo                string         `json:"logo"`
	North               string         `json:"north"`
	NumSvgImages        int            `json:"num_svg_images"`
	SmallImage          string         `json:"small_image"`
	TrackMap            string         `json:"track_map"`
	TrackMapLayers      TrackMapLayers `json:"track_map_layers"`
}

// SeriesInfo contains series information from the /data/series/get endpoint.
type SeriesInfo struct {
	SeriesID       int    `json:"series_id"`
	SeriesName     string `json:"series_name"`
	SeriesShortName string `json:"series_short_name"`
	CategoryID     int    `json:"category_id"`
	Category       string `json:"category"`
	Active         bool   `json:"active"`
	Official       bool   `json:"official"`
	FixedSetup     bool   `json:"fixed_setup"`
	Logo           string `json:"logo"`
	SearchFilters  string `json:"search_filters"`
	MinStarters    int    `json:"min_starters"`
	MaxStarters    int    `json:"max_starters"`
	Oval           bool   `json:"oval"`
	Road           bool   `json:"road"`
	Dirt           bool   `json:"dirt"`
}
