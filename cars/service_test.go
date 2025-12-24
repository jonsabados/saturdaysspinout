package cars

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type getCarsCall struct {
	result []iracing.CarInfo
	err    error
}

type getCarAssetsCall struct {
	result map[int64]iracing.CarAssets
	err    error
}

func TestService_GetAll(t *testing.T) {
	testCases := []struct {
		name string

		getCarsCall      *getCarsCall
		getCarAssetsCall *getCarAssetsCall

		expectedCars []Car
		expectedErr  error
	}{
		{
			name: "success with assets",
			getCarsCall: &getCarsCall{
				result: []iracing.CarInfo{
					{
						CarID:                1,
						CarName:              "Mazda MX-5 Miata",
						CarNameAbbreviated:   "MX-5",
						CarMake:              "Mazda",
						CarModel:             "MX-5 Miata",
						CarWeight:            2332,
						HPUnderHood:          155,
						HPActual:             155,
						Categories:           []string{"road"},
						HasHeadlights:        true,
						HasMultipleDryTires:  false,
						RainEnabled:          true,
						FreeWithSubscription: true,
						Retired:              false,
					},
					{
						CarID:                2,
						CarName:              "NASCAR Cup Series Chevrolet Camaro ZL1",
						CarNameAbbreviated:   "Cup Camaro",
						CarMake:              "Chevrolet",
						CarModel:             "Camaro ZL1",
						CarWeight:            3200,
						HPUnderHood:          670,
						HPActual:             670,
						Categories:           []string{"oval"},
						HasHeadlights:        false,
						HasMultipleDryTires:  true,
						RainEnabled:          false,
						FreeWithSubscription: false,
						Retired:              false,
					},
				},
			},
			getCarAssetsCall: &getCarAssetsCall{
				result: map[int64]iracing.CarAssets{
					1: {
						CarID:      1,
						DetailCopy: "<p>The perfect starter car</p>",
						Logo:       "/img/logos/cars/mazda-logo.png",
						Folder:     "/img/cars/mazda",
						SmallImage: "mx5-small.jpg",
						LargeImage: "mx5-large.jpg",
					},
				},
			},
			expectedCars: []Car{
				{
					ID:                   1,
					Name:                 "Mazda MX-5 Miata",
					NameAbbreviated:      "MX-5",
					Make:                 "Mazda",
					Model:                "MX-5 Miata",
					Description:          "<p>The perfect starter car</p>",
					Weight:               2332,
					HPUnderHood:          155,
					HPActual:             155,
					Categories:           []string{"road"},
					LogoURL:              "https://images-static.iracing.com/img/logos/cars/mazda-logo.png",
					SmallImageURL:        "https://images-static.iracing.com/img/cars/mazda/mx5-small.jpg",
					LargeImageURL:        "https://images-static.iracing.com/img/cars/mazda/mx5-large.jpg",
					HasHeadlights:        true,
					HasMultipleDryTires:  false,
					RainEnabled:          true,
					FreeWithSubscription: true,
					Retired:              false,
				},
				{
					ID:                   2,
					Name:                 "NASCAR Cup Series Chevrolet Camaro ZL1",
					NameAbbreviated:      "Cup Camaro",
					Make:                 "Chevrolet",
					Model:                "Camaro ZL1",
					Description:          "",
					Weight:               3200,
					HPUnderHood:          670,
					HPActual:             670,
					Categories:           []string{"oval"},
					LogoURL:              "",
					SmallImageURL:        "",
					LargeImageURL:        "",
					HasHeadlights:        false,
					HasMultipleDryTires:  true,
					RainEnabled:          false,
					FreeWithSubscription: false,
					Retired:              false,
				},
			},
		},
		{
			name: "empty cars",
			getCarsCall: &getCarsCall{
				result: []iracing.CarInfo{},
			},
			getCarAssetsCall: &getCarAssetsCall{
				result: map[int64]iracing.CarAssets{},
			},
			expectedCars: []Car{},
		},
		{
			name: "GetCars error",
			getCarsCall: &getCarsCall{
				err: errors.New("iracing API error"),
			},
			expectedErr: errors.New("iracing API error"),
		},
		{
			name: "GetCarAssets error",
			getCarsCall: &getCarsCall{
				result: []iracing.CarInfo{
					{CarID: 1, CarName: "Test Car"},
				},
			},
			getCarAssetsCall: &getCarAssetsCall{
				err: errors.New("iracing assets API error"),
			},
			expectedErr: errors.New("iracing assets API error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := NewMockIRacingClient(t)

			if tc.getCarsCall != nil {
				mockClient.EXPECT().GetCars(mock.Anything, "test-token").
					Return(tc.getCarsCall.result, tc.getCarsCall.err)
			}

			if tc.getCarAssetsCall != nil {
				mockClient.EXPECT().GetCarAssets(mock.Anything, "test-token").
					Return(tc.getCarAssetsCall.result, tc.getCarAssetsCall.err)
			}

			svc := NewService(mockClient)
			cars, err := svc.GetAll(context.Background(), "test-token")

			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedCars, cars)
		})
	}
}