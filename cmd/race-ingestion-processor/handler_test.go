package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jonsabados/saturdaysspinout/ingestion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewHandler(t *testing.T) {
	type ingestRacesCall struct {
		request ingestion.RaceIngestionRequest
		err     error
	}

	testCases := []struct {
		name              string
		messages          []events.SQSMessage
		ingestRacesCalls  []ingestRacesCall
		expectErr         bool
		expectErrContains string
	}{
		{
			name:     "empty event returns nil",
			messages: []events.SQSMessage{},
		},
		{
			name: "single valid message processes successfully",
			messages: []events.SQSMessage{
				{
					MessageId: "msg-1",
					Body:      mustJSON(ingestion.RaceIngestionRequest{DriverID: 1001, IRacingAccessToken: "token-1", NotifyConnectionID: "conn-1"}),
				},
			},
			ingestRacesCalls: []ingestRacesCall{
				{request: ingestion.RaceIngestionRequest{DriverID: 1001, IRacingAccessToken: "token-1", NotifyConnectionID: "conn-1"}},
			},
		},
		{
			name: "multiple valid messages process successfully",
			messages: []events.SQSMessage{
				{
					MessageId: "msg-1",
					Body:      mustJSON(ingestion.RaceIngestionRequest{DriverID: 1001, IRacingAccessToken: "token-1", NotifyConnectionID: "conn-1"}),
				},
				{
					MessageId: "msg-2",
					Body:      mustJSON(ingestion.RaceIngestionRequest{DriverID: 1002, IRacingAccessToken: "token-2", NotifyConnectionID: "conn-2"}),
				},
			},
			ingestRacesCalls: []ingestRacesCall{
				{request: ingestion.RaceIngestionRequest{DriverID: 1001, IRacingAccessToken: "token-1", NotifyConnectionID: "conn-1"}},
				{request: ingestion.RaceIngestionRequest{DriverID: 1002, IRacingAccessToken: "token-2", NotifyConnectionID: "conn-2"}},
			},
		},
		{
			name: "invalid JSON skipped without error",
			messages: []events.SQSMessage{
				{
					MessageId: "msg-1",
					Body:      "not valid json",
				},
			},
		},
		{
			name: "invalid JSON skipped, valid message processed",
			messages: []events.SQSMessage{
				{
					MessageId: "msg-1",
					Body:      "not valid json",
				},
				{
					MessageId: "msg-2",
					Body:      mustJSON(ingestion.RaceIngestionRequest{DriverID: 1001, IRacingAccessToken: "token-1", NotifyConnectionID: "conn-1"}),
				},
			},
			ingestRacesCalls: []ingestRacesCall{
				{request: ingestion.RaceIngestionRequest{DriverID: 1001, IRacingAccessToken: "token-1", NotifyConnectionID: "conn-1"}},
			},
		},
		{
			name: "processor error returns immediately",
			messages: []events.SQSMessage{
				{
					MessageId: "msg-1",
					Body:      mustJSON(ingestion.RaceIngestionRequest{DriverID: 1001, IRacingAccessToken: "token-1", NotifyConnectionID: "conn-1"}),
				},
			},
			ingestRacesCalls: []ingestRacesCall{
				{
					request: ingestion.RaceIngestionRequest{DriverID: 1001, IRacingAccessToken: "token-1", NotifyConnectionID: "conn-1"},
					err:     errors.New("ingestion failed"),
				},
			},
			expectErr:         true,
			expectErrContains: "ingestion failed",
		},
		{
			name: "processor error on second message stops processing",
			messages: []events.SQSMessage{
				{
					MessageId: "msg-1",
					Body:      mustJSON(ingestion.RaceIngestionRequest{DriverID: 1001, IRacingAccessToken: "token-1", NotifyConnectionID: "conn-1"}),
				},
				{
					MessageId: "msg-2",
					Body:      mustJSON(ingestion.RaceIngestionRequest{DriverID: 1002, IRacingAccessToken: "token-2", NotifyConnectionID: "conn-2"}),
				},
				{
					MessageId: "msg-3",
					Body:      mustJSON(ingestion.RaceIngestionRequest{DriverID: 1003, IRacingAccessToken: "token-3", NotifyConnectionID: "conn-3"}),
				},
			},
			ingestRacesCalls: []ingestRacesCall{
				{request: ingestion.RaceIngestionRequest{DriverID: 1001, IRacingAccessToken: "token-1", NotifyConnectionID: "conn-1"}},
				{
					request: ingestion.RaceIngestionRequest{DriverID: 1002, IRacingAccessToken: "token-2", NotifyConnectionID: "conn-2"},
					err:     errors.New("ingestion failed"),
				},
				// msg-3 not processed due to error on msg-2
			},
			expectErr:         true,
			expectErrContains: "ingestion failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockProcessor := NewMockProcessor(t)

			for _, call := range tc.ingestRacesCalls {
				mockProcessor.EXPECT().
					IngestRaces(mock.Anything, call.request).
					Return(call.err)
			}

			handler := NewHandler(mockProcessor)
			err := handler(context.Background(), events.SQSEvent{Records: tc.messages})

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}