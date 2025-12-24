package driver

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewDeleteRacesEndpoint(t *testing.T) {
	type storeCall struct {
		driverID int64
		err      error
	}

	testCases := []struct {
		name string

		driverID string

		storeCalls []storeCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:     "success",
			driverID: "12345",
			storeCalls: []storeCall{
				{
					driverID: 12345,
					err:      nil,
				},
			},
			expectedStatus:      http.StatusNoContent,
			expectedBodyFixture: "",
		},
		{
			name:     "store error",
			driverID: "12345",
			storeCalls: []storeCall{
				{
					driverID: 12345,
					err:      errors.New("database error"),
				},
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/delete_races_store_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockDeleteRacesStore(t)
			for _, call := range tc.storeCalls {
				mockStore.EXPECT().DeleteDriverRaces(mock.Anything, call.driverID).
					Return(call.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Delete("/{driver_id}", NewDeleteRacesEndpoint(mockStore).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.driverID

			req, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			assert.Equal(t, tc.expectedStatus, res.StatusCode)

			if tc.expectedBodyFixture == "" {
				bodyBytes, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Empty(t, bodyBytes)
			} else {
				bodyBytes, err := io.ReadAll(res.Body)
				require.NoError(t, err)

				expectedBody, err := os.ReadFile(tc.expectedBodyFixture)
				require.NoError(t, err)

				assert.JSONEq(t, string(expectedBody), string(bodyBytes))
			}
		})
	}
}