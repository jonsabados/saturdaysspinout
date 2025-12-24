package tracks

import (
	"context"
	"errors"
	"testing"

	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type getTracksCall struct {
	result []iracing.TrackInfo
	err    error
}

type getTrackAssetsCall struct {
	result map[int64]iracing.TrackAssets
	err    error
}

func TestService_GetAll(t *testing.T) {
	testCases := []struct {
		name string

		getTracksCall      *getTracksCall
		getTrackAssetsCall *getTrackAssetsCall

		expectedTracks []Track
		expectedErr    error
	}{
		{
			name: "success with assets",
			getTracksCall: &getTracksCall{
				result: []iracing.TrackInfo{
					{
						TrackID:              1,
						TrackName:            "Lime Rock Park",
						ConfigName:           "Full Course",
						Category:             "road",
						Location:             "Lakeville, Connecticut, USA",
						CornersPerLap:        7,
						TrackConfigLength:    1.53,
						IsDirt:               false,
						IsOval:               false,
						NightLighting:        false,
						RainEnabled:          true,
						FreeWithSubscription: false,
						Retired:              false,
						PitRoadSpeedLimit:    45,
					},
					{
						TrackID:              2,
						TrackName:            "Daytona International Speedway",
						ConfigName:           "Oval",
						Category:             "oval",
						Location:             "Daytona Beach, Florida, USA",
						CornersPerLap:        4,
						TrackConfigLength:    2.5,
						IsDirt:               false,
						IsOval:               true,
						NightLighting:        true,
						RainEnabled:          false,
						FreeWithSubscription: true,
						Retired:              false,
						PitRoadSpeedLimit:    55,
					},
				},
			},
			getTrackAssetsCall: &getTrackAssetsCall{
				result: map[int64]iracing.TrackAssets{
					1: {
						TrackID:    1,
						DetailCopy: "<p>A great road course</p>",
						TrackMap:   "https://example.com/map1",
						Logo:       "/img/logos/tracks/limerock-logo.png",
						Folder:     "/img/tracks/limerock",
						SmallImage: "limerock-small.jpg",
						LargeImage: "limerock-large.jpg",
					},
				},
			},
			expectedTracks: []Track{
				{
					ID:                1,
					Name:              "Lime Rock Park",
					ConfigName:        "Full Course",
					Category:          "road",
					Location:          "Lakeville, Connecticut, USA",
					CornersPerLap:     7,
					LengthMiles:       1.53,
					Description:       "<p>A great road course</p>",
					LogoURL:           "https://images-static.iracing.com/img/logos/tracks/limerock-logo.png",
					SmallImageURL:     "https://images-static.iracing.com/img/tracks/limerock/limerock-small.jpg",
					LargeImageURL:     "https://images-static.iracing.com/img/tracks/limerock/limerock-large.jpg",
					TrackMapURL:       "https://example.com/map1",
					IsDirt:            false,
					IsOval:            false,
					HasNightLighting:  false,
					RainEnabled:       true,
					FreeWithSub:       false,
					Retired:           false,
					PitRoadSpeedLimit: 45,
				},
				{
					ID:                2,
					Name:              "Daytona International Speedway",
					ConfigName:        "Oval",
					Category:          "oval",
					Location:          "Daytona Beach, Florida, USA",
					CornersPerLap:     4,
					LengthMiles:       2.5,
					Description:       "",
					LogoURL:           "",
					SmallImageURL:     "",
					LargeImageURL:     "",
					TrackMapURL:       "",
					IsDirt:            false,
					IsOval:            true,
					HasNightLighting:  true,
					RainEnabled:       false,
					FreeWithSub:       true,
					Retired:           false,
					PitRoadSpeedLimit: 55,
				},
			},
		},
		{
			name: "empty tracks",
			getTracksCall: &getTracksCall{
				result: []iracing.TrackInfo{},
			},
			getTrackAssetsCall: &getTrackAssetsCall{
				result: map[int64]iracing.TrackAssets{},
			},
			expectedTracks: []Track{},
		},
		{
			name: "GetTracks error",
			getTracksCall: &getTracksCall{
				err: errors.New("iracing API error"),
			},
			expectedErr: errors.New("iracing API error"),
		},
		{
			name: "GetTrackAssets error",
			getTracksCall: &getTracksCall{
				result: []iracing.TrackInfo{
					{TrackID: 1, TrackName: "Test Track"},
				},
			},
			getTrackAssetsCall: &getTrackAssetsCall{
				err: errors.New("iracing assets API error"),
			},
			expectedErr: errors.New("iracing assets API error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := NewMockIRacingClient(t)

			if tc.getTracksCall != nil {
				mockClient.EXPECT().GetTracks(mock.Anything, "test-token").
					Return(tc.getTracksCall.result, tc.getTracksCall.err)
			}

			if tc.getTrackAssetsCall != nil {
				mockClient.EXPECT().GetTrackAssets(mock.Anything, "test-token").
					Return(tc.getTrackAssetsCall.result, tc.getTrackAssetsCall.err)
			}

			svc := NewService(mockClient)
			tracks, err := svc.GetAll(context.Background(), "test-token")

			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedTracks, tracks)
		})
	}
}