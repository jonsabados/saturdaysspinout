package ingestion

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/jonsabados/saturdaysspinout/ingestion"
	"github.com/jonsabados/saturdaysspinout/store"
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

func TestNewRaceIngestionEndpoint(t *testing.T) {
	testSessionClaims := &auth.SessionClaims{
		IRacingUserID:   1100750,
		IRacingUserName: "Jon Sabados",
	}
	testSensitiveClaims := &auth.SensitiveClaims{
		IRacingAccessToken: "test-access-token",
	}

	type getDriverCall struct {
		driverID int64
		driver   *store.Driver
		err      error
	}

	type publishEventCall struct {
		event ingestion.RaceIngestionRequest
		err   error
	}

	testCases := []struct {
		name string

		sessionClaims   *auth.SessionClaims
		sensitiveClaims *auth.SensitiveClaims
		tokenErr        error
		requestBody     string

		getDriverCall     *getDriverCall
		publishEventCall  *publishEventCall

		expectedResponseStatus      int
		expectedResponseBodyFixture string
	}{
		{
			name:                        "missing authorization header returns 401",
			sessionClaims:               nil,
			sensitiveClaims:             nil,
			tokenErr:                    errors.New("missing token"),
			requestBody:                 `{"notifyConnectionId": "conn-123"}`,
			expectedResponseStatus:      http.StatusUnauthorized,
			expectedResponseBodyFixture: "fixtures/race_endpoint_unauthorized_response.json",
		},
		{
			name:            "store error returns 500",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			requestBody:     `{"notifyConnectionId": "conn-123"}`,
			getDriverCall: &getDriverCall{
				driverID: 1100750,
				driver:   nil,
				err:      errors.New("database error"),
			},
			expectedResponseStatus:      http.StatusInternalServerError,
			expectedResponseBodyFixture: "fixtures/race_endpoint_store_error_response.json",
		},
		{
			name:            "driver not found returns 404",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			requestBody:     `{"notifyConnectionId": "conn-123"}`,
			getDriverCall: &getDriverCall{
				driverID: 1100750,
				driver:   nil,
				err:      nil,
			},
			expectedResponseStatus:      http.StatusNotFound,
			expectedResponseBodyFixture: "fixtures/race_endpoint_not_found_response.json",
		},
		{
			name:            "active lock returns 429",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			requestBody:     `{"notifyConnectionId": "conn-123"}`,
			getDriverCall: &getDriverCall{
				driverID: 1100750,
				driver: &store.Driver{
					DriverID:              1100750,
					IngestionBlockedUntil: ptrTo(time.Now().Add(500 * time.Millisecond)),
				},
				err: nil,
			},
			expectedResponseStatus:      http.StatusTooManyRequests,
			expectedResponseBodyFixture: "fixtures/race_endpoint_too_many_requests_response.json",
		},
		{
			name:            "expired lock continues",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			requestBody:     `{"notifyConnectionId": "conn-123"}`,
			getDriverCall: &getDriverCall{
				driverID: 1100750,
				driver: &store.Driver{
					DriverID:              1100750,
					IngestionBlockedUntil: ptrTo(time.Now().Add(-1 * time.Minute)),
				},
				err: nil,
			},
			publishEventCall: &publishEventCall{
				event: ingestion.RaceIngestionRequest{
					DriverID:           1100750,
					IRacingAccessToken: "test-access-token",
					NotifyConnectionID: "conn-123",
				},
				err: nil,
			},
			expectedResponseStatus:      http.StatusAccepted,
			expectedResponseBodyFixture: "fixtures/race_endpoint_accepted_response.json",
		},
		{
			name:            "invalid request body returns 400",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			requestBody:     `not valid json`,
			getDriverCall: &getDriverCall{
				driverID: 1100750,
				driver:   &store.Driver{DriverID: 1100750},
				err:      nil,
			},
			expectedResponseStatus:      http.StatusBadRequest,
			expectedResponseBodyFixture: "fixtures/race_endpoint_invalid_body_response.json",
		},
		{
			name:            "missing notifyConnectionId returns 400",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			requestBody:     `{"notifyConnectionId": ""}`,
			getDriverCall: &getDriverCall{
				driverID: 1100750,
				driver:   &store.Driver{DriverID: 1100750},
				err:      nil,
			},
			expectedResponseStatus:      http.StatusBadRequest,
			expectedResponseBodyFixture: "fixtures/race_endpoint_missing_connection_id_response.json",
		},
		{
			name:            "dispatcher error returns 500",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			requestBody:     `{"notifyConnectionId": "conn-123"}`,
			getDriverCall: &getDriverCall{
				driverID: 1100750,
				driver:   &store.Driver{DriverID: 1100750},
				err:      nil,
			},
			publishEventCall: &publishEventCall{
				event: ingestion.RaceIngestionRequest{
					DriverID:           1100750,
					IRacingAccessToken: "test-access-token",
					NotifyConnectionID: "conn-123",
				},
				err: errors.New("SQS error"),
			},
			expectedResponseStatus:      http.StatusInternalServerError,
			expectedResponseBodyFixture: "fixtures/race_endpoint_dispatcher_error_response.json",
		},
		{
			name:            "success returns 202",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			requestBody:     `{"notifyConnectionId": "conn-123"}`,
			getDriverCall: &getDriverCall{
				driverID: 1100750,
				driver:   &store.Driver{DriverID: 1100750},
				err:      nil,
			},
			publishEventCall: &publishEventCall{
				event: ingestion.RaceIngestionRequest{
					DriverID:           1100750,
					IRacingAccessToken: "test-access-token",
					NotifyConnectionID: "conn-123",
				},
				err: nil,
			},
			expectedResponseStatus:      http.StatusAccepted,
			expectedResponseBodyFixture: "fixtures/race_endpoint_accepted_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			validator := &stubTokenValidator{
				sessionClaims:   tc.sessionClaims,
				sensitiveClaims: tc.sensitiveClaims,
				err:             tc.tokenErr,
			}

			mockStore := NewMockStore(t)
			if tc.getDriverCall != nil {
				mockStore.EXPECT().GetDriver(mock.Anything, tc.getDriverCall.driverID).Return(tc.getDriverCall.driver, tc.getDriverCall.err)
			}

			mockDispatcher := NewMockEventDispatcher(t)
			if tc.publishEventCall != nil {
				mockDispatcher.EXPECT().PublishEvent(mock.Anything, tc.publishEventCall.event).Return(tc.publishEventCall.err)
			}

			endpoint := NewRaceIngestionEndpoint(mockStore, mockDispatcher)
			handler := correlation.Middleware(func() string { return testCorrelationID })(api.AuthMiddleware(validator)(endpoint))

			ts := httptest.NewServer(handler)
			defer ts.Close()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, bytes.NewBufferString(tc.requestBody))
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()
			bodyBytes, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedResponseStatus, res.StatusCode)

			expectedBody, err := os.ReadFile(tc.expectedResponseBodyFixture)
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedBody), string(bodyBytes))
		})
	}
}

func ptrTo[T any](v T) *T {
	return &v
}