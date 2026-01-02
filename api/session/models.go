package session

import (
	"time"

	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/store"
)

// SessionResponse is the API response for session results.
type SessionResponse struct {
	SubsessionID            int64              `json:"subsessionId"`
	DriverRaceId            *int64             `json:"driverRaceId,omitempty"`
	SessionID               int64              `json:"sessionId"`
	AllowedLicenses         []AllowedLicense   `json:"allowedLicenses"`
	AssociatedSubsessionIDs []int64            `json:"associatedSubsessionIds"`
	CanProtest              bool               `json:"canProtest"`
	CarClasses              []CarClass         `json:"carClasses"`
	CautionType             int                `json:"cautionType"`
	CooldownMinutes         int                `json:"cooldownMinutes"`
	CornersPerLap           int                `json:"cornersPerLap"`
	DamageModel             int                `json:"damageModel"`
	DriverChangeParam1      int                `json:"driverChangeParam1"`
	DriverChangeParam2      int                `json:"driverChangeParam2"`
	DriverChangeRule        int                `json:"driverChangeRule"`
	DriverChanges           bool               `json:"driverChanges"`
	EndTime                 time.Time          `json:"endTime"`
	EventAverageLap         int                `json:"eventAverageLap"`
	EventBestLapTime        int                `json:"eventBestLapTime"`
	EventLapsComplete       int                `json:"eventLapsComplete"`
	EventStrengthOfField    int                `json:"eventStrengthOfField"`
	EventType               int                `json:"eventType"`
	EventTypeName           string             `json:"eventTypeName"`
	HeatInfoID              int                `json:"heatInfoId"`
	LicenseCategory         string             `json:"licenseCategory"`
	LicenseCategoryID       int                `json:"licenseCategoryId"`
	LimitMinutes            int                `json:"limitMinutes"`
	MaxTeamDrivers          int                `json:"maxTeamDrivers"`
	MaxWeeks                int                `json:"maxWeeks"`
	MinTeamDrivers          int                `json:"minTeamDrivers"`
	NumCautionLaps          int                `json:"numCautionLaps"`
	NumCautions             int                `json:"numCautions"`
	NumDrivers              int                `json:"numDrivers"`
	NumLapsForQualAverage   int                `json:"numLapsForQualAverage"`
	NumLapsForSoloAverage   int                `json:"numLapsForSoloAverage"`
	NumLeadChanges          int                `json:"numLeadChanges"`
	OfficialSession         bool               `json:"officialSession"`
	PointsType              string             `json:"pointsType"`
	PrivateSessionID        int                `json:"privateSessionId"`
	RaceSummary             RaceSummary        `json:"raceSummary"`
	RaceWeekNum             int                `json:"raceWeekNum"`
	ResultsRestricted       bool               `json:"resultsRestricted"`
	SeasonID                int                `json:"seasonId"`
	SeasonName              string             `json:"seasonName"`
	SeasonQuarter           int                `json:"seasonQuarter"`
	SeasonShortName         string             `json:"seasonShortName"`
	SeasonYear              int                `json:"seasonYear"`
	SeriesID                int                `json:"seriesId"`
	SeriesLogo              string             `json:"seriesLogo"`
	SeriesName              string             `json:"seriesName"`
	SeriesShortName         string             `json:"seriesShortName"`
	SessionResults          []SimSessionResult `json:"sessionResults"`
	SessionSplits           []SessionSplit     `json:"sessionSplits"`
	SpecialEventType        int                `json:"specialEventType"`
	StartTime               time.Time          `json:"startTime"`
	Track                   Track              `json:"track"`
	TrackState              TrackState         `json:"trackState"`
	Weather                 Weather            `json:"weather"`
}

type AllowedLicense struct {
	GroupName       string `json:"groupName"`
	LicenseGroup    int    `json:"licenseGroup"`
	MaxLicenseLevel int    `json:"maxLicenseLevel"`
	MinLicenseLevel int    `json:"minLicenseLevel"`
	ParentID        int    `json:"parentId"`
}

type CarClass struct {
	CarClassID      int          `json:"carClassId"`
	ShortName       string       `json:"shortName"`
	Name            string       `json:"name"`
	StrengthOfField int          `json:"strengthOfField"`
	NumEntries      int          `json:"numEntries"`
	CarsInClass     []CarInClass `json:"carsInClass"`
}

type CarInClass struct {
	CarID int `json:"carId"`
}

type RaceSummary struct {
	SubsessionID         int64  `json:"subsessionId"`
	AverageLap           int    `json:"averageLap"`
	LapsComplete         int    `json:"lapsComplete"`
	NumCautions          int    `json:"numCautions"`
	NumCautionLaps       int    `json:"numCautionLaps"`
	NumLeadChanges       int    `json:"numLeadChanges"`
	FieldStrength        int    `json:"fieldStrength"`
	NumOptLaps           int    `json:"numOptLaps"`
	HasOptPath           bool   `json:"hasOptPath"`
	SpecialEventType     int    `json:"specialEventType"`
	SpecialEventTypeText string `json:"specialEventTypeText"`
}

type SimSessionResult struct {
	SimsessionNumber   int            `json:"simsessionNumber"`
	SimsessionName     string         `json:"simsessionName"`
	SimsessionType     int            `json:"simsessionType"`
	SimsessionTypeName string         `json:"simsessionTypeName"`
	SimsessionSubtype  int            `json:"simsessionSubtype"`
	WeatherResult      WeatherResult  `json:"weatherResult"`
	Results            []DriverResult `json:"results"`
}

type WeatherResult struct {
	AvgSkies                 int     `json:"avgSkies"`
	AvgCloudCoverPct         float64 `json:"avgCloudCoverPct"`
	MinCloudCoverPct         float64 `json:"minCloudCoverPct"`
	MaxCloudCoverPct         float64 `json:"maxCloudCoverPct"`
	TempUnits                int     `json:"tempUnits"`
	AvgTemp                  float64 `json:"avgTemp"`
	MinTemp                  float64 `json:"minTemp"`
	MaxTemp                  float64 `json:"maxTemp"`
	AvgRelHumidity           float64 `json:"avgRelHumidity"`
	WindUnits                int     `json:"windUnits"`
	AvgWindSpeed             float64 `json:"avgWindSpeed"`
	MinWindSpeed             float64 `json:"minWindSpeed"`
	MaxWindSpeed             float64 `json:"maxWindSpeed"`
	AvgWindDir               int     `json:"avgWindDir"`
	MaxFog                   float64 `json:"maxFog"`
	FogTimePct               float64 `json:"fogTimePct"`
	PrecipTimePct            float64 `json:"precipTimePct"`
	PrecipMM                 float64 `json:"precipMm"`
	PrecipMM2HrBeforeSession float64 `json:"precipMm2hrBeforeSession"`
	SimulatedStartTime       string  `json:"simulatedStartTime"`
}

type DriverResult struct {
	CustID                  int64     `json:"custId"`
	DisplayName             string    `json:"displayName"`
	AggregateChampPoints    int       `json:"aggregateChampPoints"`
	AI                      bool      `json:"ai"`
	AverageLap              int       `json:"averageLap"`
	BestLapNum              int       `json:"bestLapNum"`
	BestLapTime             int       `json:"bestLapTime"`
	BestNLapsNum            int       `json:"bestNlapsNum"`
	BestNLapsTime           int       `json:"bestNlapsTime"`
	BestQualLapAt           time.Time `json:"bestQualLapAt"`
	BestQualLapNum          int       `json:"bestQualLapNum"`
	BestQualLapTime         int       `json:"bestQualLapTime"`
	CarClassID              int       `json:"carClassId"`
	CarClassName            string    `json:"carClassName"`
	CarClassShortName       string    `json:"carClassShortName"`
	CarID                   int64     `json:"carId"`
	CarName                 string    `json:"carName"`
	CarCfg                  int       `json:"carCfg"`
	ChampPoints             int       `json:"champPoints"`
	ClassInterval           int       `json:"classInterval"`
	CountryCode             string    `json:"countryCode"`
	Division                int       `json:"division"`
	DivisionName            string    `json:"divisionName"`
	DropRace                bool      `json:"dropRace"`
	FinishPosition          int       `json:"finishPosition"`
	FinishPositionInClass   int       `json:"finishPositionInClass"`
	FlairID                 int       `json:"flairId"`
	FlairName               string    `json:"flairName"`
	FlairShortname          string    `json:"flairShortname"`
	Friend                  bool      `json:"friend"`
	Helmet                  Helmet    `json:"helmet"`
	Incidents               int       `json:"incidents"`
	Interval                int       `json:"interval"`
	LapsComplete            int       `json:"lapsComplete"`
	LapsLead                int       `json:"lapsLead"`
	LeagueAggPoints         int       `json:"leagueAggPoints"`
	LeaguePoints            int       `json:"leaguePoints"`
	LicenseChangeOval       int       `json:"licenseChangeOval"`
	LicenseChangeRoad       int       `json:"licenseChangeRoad"`
	Livery                  Livery    `json:"livery"`
	MaxPctFuelFill          int       `json:"maxPctFuelFill"`
	NewCPI                  float64   `json:"newCpi"`
	NewLicenseLevel         int       `json:"newLicenseLevel"`
	NewSubLevel             int       `json:"newSubLevel"`
	NewTTRating             int       `json:"newTtrating"`
	NewIRating              int       `json:"newIrating"`
	OldCPI                  float64   `json:"oldCpi"`
	OldLicenseLevel         int       `json:"oldLicenseLevel"`
	OldSubLevel             int       `json:"oldSubLevel"`
	OldTTRating             int       `json:"oldTtrating"`
	OldIRating              int       `json:"oldIrating"`
	OptLapsComplete         int       `json:"optLapsComplete"`
	Position                int       `json:"position"`
	QualLapTime             int       `json:"qualLapTime"`
	ReasonOut               string    `json:"reasonOut"`
	ReasonOutID             int       `json:"reasonOutId"`
	StartingPosition        int       `json:"startingPosition"`
	StartingPositionInClass int       `json:"startingPositionInClass"`
	Suit                    Suit      `json:"suit"`
	Watched                 bool      `json:"watched"`
	WeightPenaltyKg         int       `json:"weightPenaltyKg"`
}

type Helmet struct {
	Pattern    int    `json:"pattern"`
	Color1     string `json:"color1"`
	Color2     string `json:"color2"`
	Color3     string `json:"color3"`
	FaceType   int    `json:"faceType"`
	HelmetType int    `json:"helmetType"`
}

type Livery struct {
	CarID        int     `json:"carId"`
	Pattern      int     `json:"pattern"`
	Color1       string  `json:"color1"`
	Color2       string  `json:"color2"`
	Color3       string  `json:"color3"`
	NumberFont   int     `json:"numberFont"`
	NumberColor1 string  `json:"numberColor1"`
	NumberColor2 string  `json:"numberColor2"`
	NumberColor3 string  `json:"numberColor3"`
	NumberSlant  int     `json:"numberSlant"`
	Sponsor1     int     `json:"sponsor1"`
	Sponsor2     int     `json:"sponsor2"`
	CarNumber    string  `json:"carNumber"`
	WheelColor   *string `json:"wheelColor"`
	RimType      int     `json:"rimType"`
}

type Suit struct {
	Pattern int    `json:"pattern"`
	Color1  string `json:"color1"`
	Color2  string `json:"color2"`
	Color3  string `json:"color3"`
}

type SessionSplit struct {
	SubsessionID         int64 `json:"subsessionId"`
	EventStrengthOfField int   `json:"eventStrengthOfField"`
}

type Track struct {
	TrackID    int64  `json:"trackId"`
	TrackName  string `json:"trackName"`
	ConfigName string `json:"configName"`
	Category   string `json:"category"`
	CategoryID int    `json:"categoryId"`
}

type TrackState struct {
	LeaveMarbles   bool `json:"leaveMarbles"`
	PracticeRubber int  `json:"practiceRubber"`
	QualifyRubber  int  `json:"qualifyRubber"`
	RaceRubber     int  `json:"raceRubber"`
	WarmupRubber   int  `json:"warmupRubber"`
}

type Weather struct {
	AllowFog                      bool    `json:"allowFog"`
	Fog                           int     `json:"fog"`
	PrecipMM2HrBeforeFinalSession float64 `json:"precipMm2hrBeforeFinalSession"`
	PrecipMMFinalSession          float64 `json:"precipMmFinalSession"`
	PrecipOption                  int     `json:"precipOption"`
	PrecipTimePct                 float64 `json:"precipTimePct"`
	RelHumidity                   int     `json:"relHumidity"`
	SimulatedStartTime            string  `json:"simulatedStartTime"`
	Skies                         int     `json:"skies"`
	TempUnits                     int     `json:"tempUnits"`
	TempValue                     int     `json:"tempValue"`
	TimeOfDay                     int     `json:"timeOfDay"`
	TrackWater                    int     `json:"trackWater"`
	Type                          int     `json:"type"`
	Version                       int     `json:"version"`
	WeatherVarInitial             int     `json:"weatherVarInitial"`
	WeatherVarOngoing             int     `json:"weatherVarOngoing"`
	WindDir                       int     `json:"windDir"`
	WindUnits                     int     `json:"windUnits"`
	WindValue                     int     `json:"windValue"`
}

// isDriverInSession checks if the given driver ID is present in any session result.
func isDriverInSession(sr *iracing.SessionResult, driverID int64) bool {
	for _, ssr := range sr.SessionResults {
		for _, dr := range ssr.Results {
			if dr.CustID == driverID {
				return true
			}
		}
	}
	return false
}

func sessionResponseFromIRacing(sr *iracing.SessionResult, currentDriverID int64) SessionResponse {
	allowedLicenses := make([]AllowedLicense, len(sr.AllowedLicenses))
	for i, al := range sr.AllowedLicenses {
		allowedLicenses[i] = AllowedLicense{
			GroupName:       al.GroupName,
			LicenseGroup:    al.LicenseGroup,
			MaxLicenseLevel: al.MaxLicenseLevel,
			MinLicenseLevel: al.MinLicenseLevel,
			ParentID:        al.ParentID,
		}
	}

	carClasses := make([]CarClass, len(sr.CarClasses))
	for i, cc := range sr.CarClasses {
		carsInClass := make([]CarInClass, len(cc.CarsInClass))
		for j, c := range cc.CarsInClass {
			carsInClass[j] = CarInClass{CarID: c.CarID}
		}
		carClasses[i] = CarClass{
			CarClassID:      cc.CarClassID,
			ShortName:       cc.ShortName,
			Name:            cc.Name,
			StrengthOfField: cc.StrengthOfField,
			NumEntries:      cc.NumEntries,
			CarsInClass:     carsInClass,
		}
	}

	sessionResults := make([]SimSessionResult, len(sr.SessionResults))
	for i, ssr := range sr.SessionResults {
		results := make([]DriverResult, len(ssr.Results))
		for j, dr := range ssr.Results {
			results[j] = driverResultFromIRacing(dr)
		}
		sessionResults[i] = SimSessionResult{
			SimsessionNumber:   ssr.SimsessionNumber,
			SimsessionName:     ssr.SimsessionName,
			SimsessionType:     ssr.SimsessionType,
			SimsessionTypeName: ssr.SimsessionTypeName,
			SimsessionSubtype:  ssr.SimsessionSubtype,
			WeatherResult:      weatherResultFromIRacing(ssr.WeatherResult),
			Results:            results,
		}
	}

	sessionSplits := make([]SessionSplit, len(sr.SessionSplits))
	for i, ss := range sr.SessionSplits {
		sessionSplits[i] = SessionSplit{
			SubsessionID:         ss.SubsessionID,
			EventStrengthOfField: ss.EventStrengthOfField,
		}
	}

	var driverRaceId *int64
	if isDriverInSession(sr, currentDriverID) {
		id := store.DriverRaceIDFromTime(sr.StartTime)
		driverRaceId = &id
	}

	return SessionResponse{
		SubsessionID:            sr.SubsessionID,
		DriverRaceId:            driverRaceId,
		SessionID:               sr.SessionID,
		AllowedLicenses:         allowedLicenses,
		AssociatedSubsessionIDs: sr.AssociatedSubsessionIDs,
		CanProtest:              sr.CanProtest,
		CarClasses:              carClasses,
		CautionType:             sr.CautionType,
		CooldownMinutes:         sr.CooldownMinutes,
		CornersPerLap:           sr.CornersPerLap,
		DamageModel:             sr.DamageModel,
		DriverChangeParam1:      sr.DriverChangeParam1,
		DriverChangeParam2:      sr.DriverChangeParam2,
		DriverChangeRule:        sr.DriverChangeRule,
		DriverChanges:           sr.DriverChanges,
		EndTime:                 sr.EndTime.UTC(),
		EventAverageLap:         sr.EventAverageLap,
		EventBestLapTime:        sr.EventBestLapTime,
		EventLapsComplete:       sr.EventLapsComplete,
		EventStrengthOfField:    sr.EventStrengthOfField,
		EventType:               sr.EventType,
		EventTypeName:           sr.EventTypeName,
		HeatInfoID:              sr.HeatInfoID,
		LicenseCategory:         sr.LicenseCategory,
		LicenseCategoryID:       sr.LicenseCategoryID,
		LimitMinutes:            sr.LimitMinutes,
		MaxTeamDrivers:          sr.MaxTeamDrivers,
		MaxWeeks:                sr.MaxWeeks,
		MinTeamDrivers:          sr.MinTeamDrivers,
		NumCautionLaps:          sr.NumCautionLaps,
		NumCautions:             sr.NumCautions,
		NumDrivers:              sr.NumDrivers,
		NumLapsForQualAverage:   sr.NumLapsForQualAverage,
		NumLapsForSoloAverage:   sr.NumLapsForSoloAverage,
		NumLeadChanges:          sr.NumLeadChanges,
		OfficialSession:         sr.OfficialSession,
		PointsType:              sr.PointsType,
		PrivateSessionID:        sr.PrivateSessionID,
		RaceSummary: RaceSummary{
			SubsessionID:         sr.RaceSummary.SubsessionID,
			AverageLap:           sr.RaceSummary.AverageLap,
			LapsComplete:         sr.RaceSummary.LapsComplete,
			NumCautions:          sr.RaceSummary.NumCautions,
			NumCautionLaps:       sr.RaceSummary.NumCautionLaps,
			NumLeadChanges:       sr.RaceSummary.NumLeadChanges,
			FieldStrength:        sr.RaceSummary.FieldStrength,
			NumOptLaps:           sr.RaceSummary.NumOptLaps,
			HasOptPath:           sr.RaceSummary.HasOptPath,
			SpecialEventType:     sr.RaceSummary.SpecialEventType,
			SpecialEventTypeText: sr.RaceSummary.SpecialEventTypeText,
		},
		RaceWeekNum:       sr.RaceWeekNum,
		ResultsRestricted: sr.ResultsRestricted,
		SeasonID:          sr.SeasonID,
		SeasonName:        sr.SeasonName,
		SeasonQuarter:     sr.SeasonQuarter,
		SeasonShortName:   sr.SeasonShortName,
		SeasonYear:        sr.SeasonYear,
		SeriesID:          sr.SeriesID,
		SeriesLogo:        sr.SeriesLogo,
		SeriesName:        sr.SeriesName,
		SeriesShortName:   sr.SeriesShortName,
		SessionResults:    sessionResults,
		SessionSplits:     sessionSplits,
		SpecialEventType:  sr.SpecialEventType,
		StartTime:         sr.StartTime.UTC(),
		Track: Track{
			TrackID:    sr.Track.TrackID,
			TrackName:  sr.Track.TrackName,
			ConfigName: sr.Track.ConfigName,
			Category:   sr.Track.Category,
			CategoryID: sr.Track.CategoryID,
		},
		TrackState: TrackState{
			LeaveMarbles:   sr.TrackState.LeaveMarbles,
			PracticeRubber: sr.TrackState.PracticeRubber,
			QualifyRubber:  sr.TrackState.QualifyRubber,
			RaceRubber:     sr.TrackState.RaceRubber,
			WarmupRubber:   sr.TrackState.WarmupRubber,
		},
		Weather: Weather{
			AllowFog:                      sr.Weather.AllowFog,
			Fog:                           sr.Weather.Fog,
			PrecipMM2HrBeforeFinalSession: sr.Weather.PrecipMM2HrBeforeFinalSession,
			PrecipMMFinalSession:          sr.Weather.PrecipMMFinalSession,
			PrecipOption:                  sr.Weather.PrecipOption,
			PrecipTimePct:                 sr.Weather.PrecipTimePct,
			RelHumidity:                   sr.Weather.RelHumidity,
			SimulatedStartTime:            sr.Weather.SimulatedStartTime,
			Skies:                         sr.Weather.Skies,
			TempUnits:                     sr.Weather.TempUnits,
			TempValue:                     sr.Weather.TempValue,
			TimeOfDay:                     sr.Weather.TimeOfDay,
			TrackWater:                    sr.Weather.TrackWater,
			Type:                          sr.Weather.Type,
			Version:                       sr.Weather.Version,
			WeatherVarInitial:             sr.Weather.WeatherVarInitial,
			WeatherVarOngoing:             sr.Weather.WeatherVarOngoing,
			WindDir:                       sr.Weather.WindDir,
			WindUnits:                     sr.Weather.WindUnits,
			WindValue:                     sr.Weather.WindValue,
		},
	}
}

func driverResultFromIRacing(dr iracing.DriverResult) DriverResult {
	return DriverResult{
		CustID:               dr.CustID,
		DisplayName:          dr.DisplayName,
		AggregateChampPoints: dr.AggregateChampPoints,
		AI:                   dr.AI,
		AverageLap:           dr.AverageLap,
		BestLapNum:           dr.BestLapNum,
		BestLapTime:          dr.BestLapTime,
		BestNLapsNum:         dr.BestNLapsNum,
		BestNLapsTime:        dr.BestNLapsTime,
		BestQualLapAt:        dr.BestQualLapAt.UTC(),
		BestQualLapNum:       dr.BestQualLapNum,
		BestQualLapTime:      dr.BestQualLapTime,
		CarClassID:           dr.CarClassID,
		CarClassName:         dr.CarClassName,
		CarClassShortName:    dr.CarClassShortName,
		CarID:                dr.CarID,
		CarName:              dr.CarName,
		CarCfg:               dr.CarCfg,
		ChampPoints:          dr.ChampPoints,
		ClassInterval:        dr.ClassInterval,
		CountryCode:          dr.CountryCode,
		Division:             dr.Division,
		DivisionName:         dr.DivisionName,
		DropRace:             dr.DropRace,
		FinishPosition:       dr.FinishPosition,
		FinishPositionInClass: dr.FinishPositionInClass,
		FlairID:               dr.FlairID,
		FlairName:             dr.FlairName,
		FlairShortname:        dr.FlairShortname,
		Friend:                dr.Friend,
		Helmet: Helmet{
			Pattern:    dr.Helmet.Pattern,
			Color1:     dr.Helmet.Color1,
			Color2:     dr.Helmet.Color2,
			Color3:     dr.Helmet.Color3,
			FaceType:   dr.Helmet.FaceType,
			HelmetType: dr.Helmet.HelmetType,
		},
		Incidents:               dr.Incidents,
		Interval:                dr.Interval,
		LapsComplete:            dr.LapsComplete,
		LapsLead:                dr.LapsLead,
		LeagueAggPoints:         dr.LeagueAggPoints,
		LeaguePoints:            dr.LeaguePoints,
		LicenseChangeOval:       dr.LicenseChangeOval,
		LicenseChangeRoad:       dr.LicenseChangeRoad,
		Livery: Livery{
			CarID:        dr.Livery.CarID,
			Pattern:      dr.Livery.Pattern,
			Color1:       dr.Livery.Color1,
			Color2:       dr.Livery.Color2,
			Color3:       dr.Livery.Color3,
			NumberFont:   dr.Livery.NumberFont,
			NumberColor1: dr.Livery.NumberColor1,
			NumberColor2: dr.Livery.NumberColor2,
			NumberColor3: dr.Livery.NumberColor3,
			NumberSlant:  dr.Livery.NumberSlant,
			Sponsor1:     dr.Livery.Sponsor1,
			Sponsor2:     dr.Livery.Sponsor2,
			CarNumber:    dr.Livery.CarNumber,
			WheelColor:   dr.Livery.WheelColor,
			RimType:      dr.Livery.RimType,
		},
		MaxPctFuelFill:          dr.MaxPctFuelFill,
		NewCPI:                  dr.NewCPI,
		NewLicenseLevel:         dr.NewLicenseLevel,
		NewSubLevel:             dr.NewSubLevel,
		NewTTRating:             dr.NewTTRating,
		NewIRating:              dr.NewIRating,
		OldCPI:                  dr.OldCPI,
		OldLicenseLevel:         dr.OldLicenseLevel,
		OldSubLevel:             dr.OldSubLevel,
		OldTTRating:             dr.OldTTRating,
		OldIRating:              dr.OldIRating,
		OptLapsComplete:         dr.OptLapsComplete,
		Position:                dr.Position,
		QualLapTime:             dr.QualLapTime,
		ReasonOut:               dr.ReasonOut,
		ReasonOutID:             dr.ReasonOutID,
		StartingPosition:        dr.StartingPosition,
		StartingPositionInClass: dr.StartingPositionInClass,
		Suit: Suit{
			Pattern: dr.Suit.Pattern,
			Color1:  dr.Suit.Color1,
			Color2:  dr.Suit.Color2,
			Color3:  dr.Suit.Color3,
		},
		Watched:         dr.Watched,
		WeightPenaltyKg: dr.WeightPenaltyKg,
	}
}

func weatherResultFromIRacing(wr iracing.WeatherResult) WeatherResult {
	return WeatherResult{
		AvgSkies:                 wr.AvgSkies,
		AvgCloudCoverPct:         wr.AvgCloudCoverPct,
		MinCloudCoverPct:         wr.MinCloudCoverPct,
		MaxCloudCoverPct:         wr.MaxCloudCoverPct,
		TempUnits:                wr.TempUnits,
		AvgTemp:                  wr.AvgTemp,
		MinTemp:                  wr.MinTemp,
		MaxTemp:                  wr.MaxTemp,
		AvgRelHumidity:           wr.AvgRelHumidity,
		WindUnits:                wr.WindUnits,
		AvgWindSpeed:             wr.AvgWindSpeed,
		MinWindSpeed:             wr.MinWindSpeed,
		MaxWindSpeed:             wr.MaxWindSpeed,
		AvgWindDir:               wr.AvgWindDir,
		MaxFog:                   wr.MaxFog,
		FogTimePct:               wr.FogTimePct,
		PrecipTimePct:            wr.PrecipTimePct,
		PrecipMM:                 wr.PrecipMM,
		PrecipMM2HrBeforeSession: wr.PrecipMM2HrBeforeSession,
		SimulatedStartTime:       wr.SimulatedStartTime,
	}
}

// LapDataResponse is the API response for lap data.
type LapDataResponse struct {
	BestLapNum      int       `json:"bestLapNum"`
	BestLapTime     int       `json:"bestLapTime"`
	BestNLapsNum    int       `json:"bestNlapsNum"`
	BestNLapsTime   int       `json:"bestNlapsTime"`
	BestQualLapNum  int       `json:"bestQualLapNum"`
	BestQualLapTime int       `json:"bestQualLapTime"`
	BestQualLapAt   time.Time `json:"bestQualLapAt"`
	CustID          int64     `json:"custId"`
	Name            string    `json:"name"`
	CarID           int       `json:"carId"`
	LicenseLevel    int       `json:"licenseLevel"`
	Laps            []Lap     `json:"laps"`
}

// Lap represents a single lap's timing and event data.
type Lap struct {
	LapNumber       int      `json:"lapNumber"`
	Flags           int      `json:"flags"`
	Incident        bool     `json:"incident"`
	SessionTime     int      `json:"sessionTime"`
	LapTime         int      `json:"lapTime"`
	PersonalBestLap bool     `json:"personalBestLap"`
	LapEvents       []string `json:"lapEvents"`
}

func lapDataResponseFromIRacing(ldr *iracing.LapDataResponse) LapDataResponse {
	laps := make([]Lap, len(ldr.Laps))
	for i, l := range ldr.Laps {
		laps[i] = Lap{
			LapNumber:       l.LapNumber,
			Flags:           l.Flags,
			Incident:        l.Incident,
			SessionTime:     l.SessionTime,
			LapTime:         l.LapTime,
			PersonalBestLap: l.PersonalBestLap,
			LapEvents:       l.LapEvents,
		}
	}

	var bestQualLapAt time.Time
	if ldr.BestQualLapAt != nil {
		bestQualLapAt = ldr.BestQualLapAt.UTC()
	}

	return LapDataResponse{
		BestLapNum:      ldr.BestLapNum,
		BestLapTime:     ldr.BestLapTime,
		BestNLapsNum:    ldr.BestNLapsNum,
		BestNLapsTime:   ldr.BestNLapsTime,
		BestQualLapNum:  ldr.BestQualLapNum,
		BestQualLapTime: ldr.BestQualLapTime,
		BestQualLapAt:   bestQualLapAt,
		CustID:          ldr.CustID,
		Name:            ldr.Name,
		CarID:           ldr.CarID,
		LicenseLevel:    ldr.LicenseLevel,
		Laps:            laps,
	}
}