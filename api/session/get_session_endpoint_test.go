package session

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testCorrelationID = "test-correlation-id"

type stubTokenValidator struct {
	sessionClaims   *auth.SessionClaims
	sensitiveClaims *auth.SensitiveClaims
	err             error
}

func (s *stubTokenValidator) ValidateToken(_ context.Context, _ string) (*auth.SessionClaims, *auth.SensitiveClaims, error) {
	return s.sessionClaims, s.sensitiveClaims, s.err
}

func TestNewGetSessionEndpoint(t *testing.T) {
	testSessionClaims := &auth.SessionClaims{
		IRacingUserID:   1100750,
		IRacingUserName: "Jon Sabados",
	}
	testSensitiveClaims := &auth.SensitiveClaims{
		IRacingAccessToken: "test-access-token",
	}

	startTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC)
	qualLapTime := time.Date(2024, 1, 15, 14, 25, 0, 0, time.UTC)

	// Create a session result where the user is a participant
	testSessionResult := &iracing.SessionResult{
		SubsessionID: 12345678,
		SessionID:    87654321,
		AllowedLicenses: []iracing.AllowedLicense{
			{
				GroupName:       "Class D",
				LicenseGroup:    3,
				MaxLicenseLevel: 12,
				MinLicenseLevel: 4,
				ParentID:        0,
			},
		},
		AssociatedSubsessionIDs: []int64{12345679, 12345680},
		CanProtest:              true,
		CarClasses: []iracing.CarClass{
			{
				CarClassID:      74,
				ShortName:       "MX-5",
				Name:            "Mazda MX-5 Cup",
				StrengthOfField: 1850,
				NumEntries:      20,
				CarsInClass:     []iracing.CarInClass{{CarID: 67}},
			},
		},
		CautionType:          0,
		CooldownMinutes:      0,
		CornersPerLap:        11,
		DamageModel:          2,
		DriverChangeParam1:   0,
		DriverChangeParam2:   0,
		DriverChangeRule:     0,
		DriverChanges:        false,
		EndTime:              endTime,
		EventAverageLap:      96500,
		EventBestLapTime:     95200,
		EventLapsComplete:    15,
		EventStrengthOfField: 1850,
		EventType:            5,
		EventTypeName:        "Race",
		HeatInfoID:           0,
		LicenseCategory:      "road",
		LicenseCategoryID:    2,
		LimitMinutes:         25,
		MaxTeamDrivers:       1,
		MaxWeeks:             12,
		MinTeamDrivers:       1,
		NumCautionLaps:       0,
		NumCautions:          0,
		NumDrivers:           20,
		NumLapsForQualAverage: 2,
		NumLapsForSoloAverage: 3,
		NumLeadChanges:       5,
		OfficialSession:      true,
		PointsType:           "race",
		PrivateSessionID:     0,
		RaceSummary: iracing.RaceSummary{
			SubsessionID:         12345678,
			AverageLap:           96500,
			LapsComplete:         15,
			NumCautions:          0,
			NumCautionLaps:       0,
			NumLeadChanges:       5,
			FieldStrength:        1850,
			NumOptLaps:           0,
			HasOptPath:           false,
			SpecialEventType:     0,
			SpecialEventTypeText: "",
		},
		RaceWeekNum:       3,
		ResultsRestricted: false,
		SeasonID:          4500,
		SeasonName:        "2024 Season 1",
		SeasonQuarter:     1,
		SeasonShortName:   "2024S1",
		SeasonYear:        2024,
		SeriesID:          231,
		SeriesLogo:        "series_logo.png",
		SeriesName:        "Advanced Mazda MX-5 Cup Series",
		SeriesShortName:   "AMXCS",
		SessionResults: []iracing.SimSessionResult{
			{
				SimsessionNumber:   0,
				SimsessionName:     "RACE",
				SimsessionType:     6,
				SimsessionTypeName: "Race",
				SimsessionSubtype:  0,
				WeatherResult: iracing.WeatherResult{
					AvgSkies:           1,
					AvgCloudCoverPct:   25.5,
					MinCloudCoverPct:   20.0,
					MaxCloudCoverPct:   30.0,
					TempUnits:          0,
					AvgTemp:            22.5,
					MinTemp:            21.0,
					MaxTemp:            24.0,
					AvgRelHumidity:     55.0,
					WindUnits:          0,
					AvgWindSpeed:       5.5,
					MinWindSpeed:       3.0,
					MaxWindSpeed:       8.0,
					AvgWindDir:         180,
					MaxFog:             0,
					FogTimePct:         0,
					PrecipTimePct:      0,
					PrecipMM:           0,
					PrecipMM2HrBeforeSession: 0,
					SimulatedStartTime: "2024-01-15T10:00",
				},
				Results: []iracing.DriverResult{
					{
						CustID:               1100750,
						DisplayName:          "Jon Sabados",
						AggregateChampPoints: 150,
						AI:                   false,
						AverageLap:           96800,
						BestLapNum:           8,
						BestLapTime:          95500,
						BestNLapsNum:         3,
						BestNLapsTime:        287000,
						BestQualLapAt:        qualLapTime,
						BestQualLapNum:       2,
						BestQualLapTime:      95300,
						CarClassID:           74,
						CarClassName:         "Mazda MX-5 Cup",
						CarClassShortName:    "MX-5",
						CarID:                67,
						CarName:              "Mazda MX-5 Cup",
						CarCfg:               0,
						ChampPoints:          50,
						ClassInterval:        0,
						CountryCode:          "US",
						Division:             3,
						DivisionName:         "Division 3",
						DropRace:             false,
						FinishPosition:       2,
						FinishPositionInClass: 2,
						FlairID:              0,
						FlairName:            "",
						FlairShortname:       "",
						Friend:               false,
						Helmet: iracing.Helmet{
							Pattern:    1,
							Color1:     "ffffff",
							Color2:     "000000",
							Color3:     "ff0000",
							FaceType:   0,
							HelmetType: 0,
						},
						Incidents:               4,
						Interval:                1500,
						LapsComplete:            15,
						LapsLead:                3,
						LeagueAggPoints:         0,
						LeaguePoints:            0,
						LicenseChangeOval:       0,
						LicenseChangeRoad:       15,
						Livery: iracing.Livery{
							CarID:        67,
							Pattern:      1,
							Color1:       "ff0000",
							Color2:       "ffffff",
							Color3:       "000000",
							NumberFont:   0,
							NumberColor1: "ffffff",
							NumberColor2: "000000",
							NumberColor3: "ff0000",
							NumberSlant:  0,
							Sponsor1:     0,
							Sponsor2:     0,
							CarNumber:    "42",
							WheelColor:   nil,
							RimType:      0,
						},
						MaxPctFuelFill:          100,
						NewCPI:                  1.45,
						NewLicenseLevel:         8,
						NewSubLevel:             399,
						NewTTRating:             0,
						NewIRating:              1875,
						OldCPI:                  1.42,
						OldLicenseLevel:         8,
						OldSubLevel:             385,
						OldTTRating:             0,
						OldIRating:              1850,
						OptLapsComplete:         0,
						Position:                2,
						QualLapTime:             95300,
						ReasonOut:               "Running",
						ReasonOutID:             0,
						StartingPosition:        5,
						StartingPositionInClass: 5,
						Suit: iracing.Suit{
							Pattern: 1,
							Color1:  "ff0000",
							Color2:  "ffffff",
							Color3:  "000000",
						},
						Watched:         false,
						WeightPenaltyKg: 0,
					},
				},
			},
		},
		SessionSplits: []iracing.SessionSplit{
			{
				SubsessionID:         12345678,
				EventStrengthOfField: 1850,
			},
		},
		SpecialEventType: 0,
		StartTime:        startTime,
		Track: iracing.Track{
			TrackID:    167,
			TrackName:  "Laguna Seca",
			ConfigName: "Full Course",
			Category:   "road",
			CategoryID: 2,
		},
		TrackState: iracing.TrackState{
			LeaveMarbles:   true,
			PracticeRubber: 0,
			QualifyRubber:  0,
			RaceRubber:     0,
			WarmupRubber:   0,
		},
		Weather: iracing.Weather{
			AllowFog:                      false,
			Fog:                           0,
			PrecipMM2HrBeforeFinalSession: 0,
			PrecipMMFinalSession:          0,
			PrecipOption:                  0,
			PrecipTimePct:                 0,
			RelHumidity:                   55,
			SimulatedStartTime:            "2024-01-15T10:00",
			Skies:                         1,
			TempUnits:                     0,
			TempValue:                     22,
			TimeOfDay:                     0,
			TrackWater:                    0,
			Type:                          0,
			Version:                       1,
			WeatherVarInitial:             0,
			WeatherVarOngoing:             0,
			WindDir:                       180,
			WindUnits:                     0,
			WindValue:                     5,
		},
	}

	// Create a session result where the user is NOT a participant (spectating)
	testSessionResultSpectator := &iracing.SessionResult{
		SubsessionID: 12345678,
		SessionID:    87654321,
		AllowedLicenses: []iracing.AllowedLicense{
			{
				GroupName:       "Class D",
				LicenseGroup:    3,
				MaxLicenseLevel: 12,
				MinLicenseLevel: 4,
				ParentID:        0,
			},
		},
		AssociatedSubsessionIDs: []int64{12345679, 12345680},
		CanProtest:              true,
		CarClasses: []iracing.CarClass{
			{
				CarClassID:      74,
				ShortName:       "MX-5",
				Name:            "Mazda MX-5 Cup",
				StrengthOfField: 1850,
				NumEntries:      20,
				CarsInClass:     []iracing.CarInClass{{CarID: 67}},
			},
		},
		CautionType:          0,
		CooldownMinutes:      0,
		CornersPerLap:        11,
		DamageModel:          2,
		DriverChangeParam1:   0,
		DriverChangeParam2:   0,
		DriverChangeRule:     0,
		DriverChanges:        false,
		EndTime:              endTime,
		EventAverageLap:      96500,
		EventBestLapTime:     95200,
		EventLapsComplete:    15,
		EventStrengthOfField: 1850,
		EventType:            5,
		EventTypeName:        "Race",
		HeatInfoID:           0,
		LicenseCategory:      "road",
		LicenseCategoryID:    2,
		LimitMinutes:         25,
		MaxTeamDrivers:       1,
		MaxWeeks:             12,
		MinTeamDrivers:       1,
		NumCautionLaps:       0,
		NumCautions:          0,
		NumDrivers:           20,
		NumLapsForQualAverage: 2,
		NumLapsForSoloAverage: 3,
		NumLeadChanges:       5,
		OfficialSession:      true,
		PointsType:           "race",
		PrivateSessionID:     0,
		RaceSummary: iracing.RaceSummary{
			SubsessionID:         12345678,
			AverageLap:           96500,
			LapsComplete:         15,
			NumCautions:          0,
			NumCautionLaps:       0,
			NumLeadChanges:       5,
			FieldStrength:        1850,
			NumOptLaps:           0,
			HasOptPath:           false,
			SpecialEventType:     0,
			SpecialEventTypeText: "",
		},
		RaceWeekNum:       3,
		ResultsRestricted: false,
		SeasonID:          4500,
		SeasonName:        "2024 Season 1",
		SeasonQuarter:     1,
		SeasonShortName:   "2024S1",
		SeasonYear:        2024,
		SeriesID:          231,
		SeriesLogo:        "series_logo.png",
		SeriesName:        "Advanced Mazda MX-5 Cup Series",
		SeriesShortName:   "AMXCS",
		SessionResults: []iracing.SimSessionResult{
			{
				SimsessionNumber:   0,
				SimsessionName:     "RACE",
				SimsessionType:     6,
				SimsessionTypeName: "Race",
				SimsessionSubtype:  0,
				WeatherResult: iracing.WeatherResult{
					AvgSkies:                 1,
					AvgCloudCoverPct:         25.5,
					MinCloudCoverPct:         20.0,
					MaxCloudCoverPct:         30.0,
					TempUnits:                0,
					AvgTemp:                  22.5,
					MinTemp:                  21.0,
					MaxTemp:                  24.0,
					AvgRelHumidity:           55.0,
					WindUnits:                0,
					AvgWindSpeed:             5.5,
					MinWindSpeed:             3.0,
					MaxWindSpeed:             8.0,
					AvgWindDir:               180,
					MaxFog:                   0,
					FogTimePct:               0,
					PrecipTimePct:            0,
					PrecipMM:                 0,
					PrecipMM2HrBeforeSession: 0,
					SimulatedStartTime:       "2024-01-15T10:00",
				},
				Results: []iracing.DriverResult{
					{
						CustID:                9999999, // Different driver, not the authenticated user
						DisplayName:           "Other Driver",
						AggregateChampPoints:  150,
						AI:                    false,
						AverageLap:            96800,
						BestLapNum:            8,
						BestLapTime:           95500,
						BestNLapsNum:          3,
						BestNLapsTime:         287000,
						BestQualLapAt:         qualLapTime,
						BestQualLapNum:        2,
						BestQualLapTime:       95300,
						CarClassID:            74,
						CarClassName:          "Mazda MX-5 Cup",
						CarClassShortName:     "MX-5",
						CarID:                 67,
						CarName:               "Mazda MX-5 Cup",
						CarCfg:                0,
						ChampPoints:           50,
						ClassInterval:         0,
						CountryCode:           "US",
						Division:              3,
						DivisionName:          "Division 3",
						DropRace:              false,
						FinishPosition:        2,
						FinishPositionInClass: 2,
						FlairID:               0,
						FlairName:             "",
						FlairShortname:        "",
						Friend:                false,
						Helmet: iracing.Helmet{
							Pattern:    1,
							Color1:     "ffffff",
							Color2:     "000000",
							Color3:     "ff0000",
							FaceType:   0,
							HelmetType: 0,
						},
						Incidents:         4,
						Interval:          1500,
						LapsComplete:      15,
						LapsLead:          3,
						LeagueAggPoints:   0,
						LeaguePoints:      0,
						LicenseChangeOval: 0,
						LicenseChangeRoad: 15,
						Livery: iracing.Livery{
							CarID:        67,
							Pattern:      1,
							Color1:       "ff0000",
							Color2:       "ffffff",
							Color3:       "000000",
							NumberFont:   0,
							NumberColor1: "ffffff",
							NumberColor2: "000000",
							NumberColor3: "ff0000",
							NumberSlant:  0,
							Sponsor1:     0,
							Sponsor2:     0,
							CarNumber:    "42",
							WheelColor:   nil,
							RimType:      0,
						},
						MaxPctFuelFill:          100,
						NewCPI:                  1.45,
						NewLicenseLevel:         8,
						NewSubLevel:             399,
						NewTTRating:             0,
						NewIRating:              1875,
						OldCPI:                  1.42,
						OldLicenseLevel:         8,
						OldSubLevel:             385,
						OldTTRating:             0,
						OldIRating:              1850,
						OptLapsComplete:         0,
						Position:                2,
						QualLapTime:             95300,
						ReasonOut:               "Running",
						ReasonOutID:             0,
						StartingPosition:        5,
						StartingPositionInClass: 5,
						Suit: iracing.Suit{
							Pattern: 1,
							Color1:  "ff0000",
							Color2:  "ffffff",
							Color3:  "000000",
						},
						Watched:         false,
						WeightPenaltyKg: 0,
					},
				},
			},
		},
		SessionSplits: []iracing.SessionSplit{
			{
				SubsessionID:         12345678,
				EventStrengthOfField: 1850,
			},
		},
		SpecialEventType: 0,
		StartTime:        startTime,
		Track: iracing.Track{
			TrackID:    167,
			TrackName:  "Laguna Seca",
			ConfigName: "Full Course",
			Category:   "road",
			CategoryID: 2,
		},
		TrackState: iracing.TrackState{
			LeaveMarbles:   true,
			PracticeRubber: 0,
			QualifyRubber:  0,
			RaceRubber:     0,
			WarmupRubber:   0,
		},
		Weather: iracing.Weather{
			AllowFog:                      false,
			Fog:                           0,
			PrecipMM2HrBeforeFinalSession: 0,
			PrecipMMFinalSession:          0,
			PrecipOption:                  0,
			PrecipTimePct:                 0,
			RelHumidity:                   55,
			SimulatedStartTime:            "2024-01-15T10:00",
			Skies:                         1,
			TempUnits:                     0,
			TempValue:                     22,
			TimeOfDay:                     0,
			TrackWater:                    0,
			Type:                          0,
			Version:                       1,
			WeatherVarInitial:             0,
			WeatherVarOngoing:             0,
			WindDir:                       180,
			WindUnits:                     0,
			WindValue:                     5,
		},
	}

	type clientCall struct {
		subsessionID int64
		result       *iracing.SessionResult
		err          error
	}

	testCases := []struct {
		name string

		subsessionID string

		sessionClaims   *auth.SessionClaims
		sensitiveClaims *auth.SensitiveClaims
		tokenErr        error

		clientCall *clientCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:            "success",
			subsessionID:    "12345678",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			clientCall: &clientCall{
				subsessionID: 12345678,
				result:       testSessionResult,
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_session_success_response.json",
		},
		{
			name:            "invalid subsession_id",
			subsessionID:    "not-a-number",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_session_invalid_id_response.json",
		},
		{
			name:                "unauthorized",
			subsessionID:        "12345678",
			sessionClaims:       nil,
			sensitiveClaims:     nil,
			tokenErr:            errors.New("invalid token"),
			expectedStatus:      http.StatusUnauthorized,
			expectedBodyFixture: "fixtures/get_session_unauthorized_response.json",
		},
		{
			name:            "iracing token expired",
			subsessionID:    "12345678",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			clientCall: &clientCall{
				subsessionID: 12345678,
				err:          iracing.ErrUpstreamUnauthorized,
			},
			expectedStatus:      http.StatusUnauthorized,
			expectedBodyFixture: "fixtures/get_session_iracing_expired_response.json",
		},
		{
			name:            "client error",
			subsessionID:    "12345678",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			clientCall: &clientCall{
				subsessionID: 12345678,
				err:          errors.New("iracing API error"),
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_session_error_response.json",
		},
		{
			name:            "spectator - user not in session",
			subsessionID:    "12345678",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			clientCall: &clientCall{
				subsessionID: 12345678,
				result:       testSessionResultSpectator,
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_session_spectator_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := &stubTokenValidator{
				sessionClaims:   tc.sessionClaims,
				sensitiveClaims: tc.sensitiveClaims,
				err:             tc.tokenErr,
			}

			mockClient := NewMockIRacingClient(t)
			if tc.clientCall != nil {
				mockClient.EXPECT().GetSessionResults(mock.Anything, "test-access-token", tc.clientCall.subsessionID, mock.Anything).
					Return(tc.clientCall.result, tc.clientCall.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Use(api.AuthMiddleware(validator))
			r.Get("/{"+SubsessionIDPathParam+"}", NewGetSessionEndpoint(mockClient).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.subsessionID

			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer test-token")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			bodyBytes, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, res.StatusCode)

			expectedBody, err := os.ReadFile(tc.expectedBodyFixture)
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedBody), string(bodyBytes))
		})
	}
}