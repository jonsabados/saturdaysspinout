package series

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type getSeriesCall struct {
	result []iracing.SeriesInfo
	err    error
}

func TestService_GetAll(t *testing.T) {
	testCases := []struct {
		name string

		getSeriesCall *getSeriesCall

		expectedSeries []Series
		expectedErr    error
	}{
		{
			name: "success with logo",
			getSeriesCall: &getSeriesCall{
				result: []iracing.SeriesInfo{
					{
						SeriesID:        159,
						SeriesName:      "Porsche 911 GT3 Cup",
						SeriesShortName: "Porsche Cup",
						CategoryID:      2,
						Category:        "Road",
						Active:          true,
						Official:        true,
						FixedSetup:      false,
						Logo:            "/img/logos/series/porsche-cup-logo.png",
					},
					{
						SeriesID:        236,
						SeriesName:      "NASCAR Cup Series",
						SeriesShortName: "Cup Series",
						CategoryID:      1,
						Category:        "Oval",
						Active:          true,
						Official:        true,
						FixedSetup:      true,
						Logo:            "",
					},
				},
			},
			expectedSeries: []Series{
				{
					ID:        159,
					Name:      "Porsche 911 GT3 Cup",
					ShortName: "Porsche Cup",
					Category:  "Road",
					LogoURL:   "https://images-static.iracing.com/img/logos/series/porsche-cup-logo.png",
					Active:    true,
					Official:  true,
				},
				{
					ID:        236,
					Name:      "NASCAR Cup Series",
					ShortName: "Cup Series",
					Category:  "Oval",
					LogoURL:   "",
					Active:    true,
					Official:  true,
				},
			},
		},
		{
			name: "empty series",
			getSeriesCall: &getSeriesCall{
				result: []iracing.SeriesInfo{},
			},
			expectedSeries: []Series{},
		},
		{
			name: "GetSeries error",
			getSeriesCall: &getSeriesCall{
				err: errors.New("iracing API error"),
			},
			expectedErr: errors.New("iracing API error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := NewMockIRacingClient(t)

			if tc.getSeriesCall != nil {
				mockClient.EXPECT().GetSeries(mock.Anything, "test-token").
					Return(tc.getSeriesCall.result, tc.getSeriesCall.err)
			}

			svc := NewService(mockClient)
			series, err := svc.GetAll(context.Background(), "test-token")

			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedSeries, series)
		})
	}
}