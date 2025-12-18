package ws

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPusher_Push(t *testing.T) {
	type postToConnectionCall struct {
		connectionID string
		data         []byte
		err          error
	}

	testCases := []struct {
		name         string
		connectionID string
		actionType   string
		payload      any

		postToConnectionCall postToConnectionCall

		expectedOK     bool
		expectedErrMsg string
	}{
		{
			name:         "successful push",
			connectionID: "conn-123",
			actionType:   "test-action",
			payload:      map[string]string{"key": "value"},
			postToConnectionCall: postToConnectionCall{
				connectionID: "conn-123",
				data:         mustMarshal(t, Message{Action: "test-action", Payload: map[string]string{"key": "value"}}),
			},
			expectedOK: true,
		},
		{
			name:         "successful push with nil payload",
			connectionID: "conn-456",
			actionType:   "ping",
			payload:      nil,
			postToConnectionCall: postToConnectionCall{
				connectionID: "conn-456",
				data:         mustMarshal(t, Message{Action: "ping"}),
			},
			expectedOK: true,
		},
		{
			name:         "connection gone returns false without error",
			connectionID: "conn-gone",
			actionType:   "test-action",
			payload:      "test",
			postToConnectionCall: postToConnectionCall{
				connectionID: "conn-gone",
				data:         mustMarshal(t, Message{Action: "test-action", Payload: "test"}),
				err:          &types.GoneException{Message: aws.String("connection gone")},
			},
			expectedOK: false,
		},
		{
			name:         "other error returns false with error",
			connectionID: "conn-err",
			actionType:   "test-action",
			payload:      "test",
			postToConnectionCall: postToConnectionCall{
				connectionID: "conn-err",
				data:         mustMarshal(t, Message{Action: "test-action", Payload: "test"}),
				err:          errors.New("network error"),
			},
			expectedOK:     false,
			expectedErrMsg: "network error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := NewMockAPIGatewayManagementClient(t)

			mockClient.EXPECT().PostToConnection(mock.Anything, &apigatewaymanagementapi.PostToConnectionInput{
				ConnectionId: aws.String(tc.postToConnectionCall.connectionID),
				Data:         tc.postToConnectionCall.data,
			}).Return(&apigatewaymanagementapi.PostToConnectionOutput{}, tc.postToConnectionCall.err)

			mockConnLookup := NewMockConnectionLookup(t)
			pusher := NewPusher(mockClient, mockConnLookup)
			ok, err := pusher.Push(context.Background(), tc.connectionID, tc.actionType, tc.payload)

			assert.Equal(t, tc.expectedOK, ok)
			if tc.expectedErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPusher_Push_MarshalError(t *testing.T) {
	mockClient := NewMockAPIGatewayManagementClient(t)
	mockConnLookup := NewMockConnectionLookup(t)
	pusher := NewPusher(mockClient, mockConnLookup)

	// channels cannot be marshaled to JSON
	unmarshalable := make(chan int)

	ok, err := pusher.Push(context.Background(), "conn-123", "test", unmarshalable)

	assert.False(t, ok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json")
}

func TestPusher_Disconnect(t *testing.T) {
	type deleteConnectionCall struct {
		connectionID string
		err          error
	}

	testCases := []struct {
		name         string
		connectionID string

		deleteConnectionCall deleteConnectionCall
	}{
		{
			name:                 "successful disconnect",
			connectionID:         "conn-123",
			deleteConnectionCall: deleteConnectionCall{connectionID: "conn-123"},
		},
		{
			name:                 "disconnect error is logged but not returned",
			connectionID:         "conn-456",
			deleteConnectionCall: deleteConnectionCall{connectionID: "conn-456", err: errors.New("disconnect failed")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := NewMockAPIGatewayManagementClient(t)

			mockClient.EXPECT().DeleteConnection(mock.Anything, &apigatewaymanagementapi.DeleteConnectionInput{
				ConnectionId: aws.String(tc.deleteConnectionCall.connectionID),
			}).Return(&apigatewaymanagementapi.DeleteConnectionOutput{}, tc.deleteConnectionCall.err)

			mockConnLookup := NewMockConnectionLookup(t)
			pusher := NewPusher(mockClient, mockConnLookup)

			logger := zerolog.Nop()
			ctx := logger.WithContext(context.Background())

			pusher.Disconnect(ctx, tc.connectionID)
		})
	}
}

func TestPusher_Broadcast(t *testing.T) {
	driverID := int64(12345)

	type getConnectionsByDriverCall struct {
		driverID int64
		result   []store.WebSocketConnection
		err      error
	}

	type postToConnectionCall struct {
		connectionID string
		data         []byte
		err          error
	}

	testCases := []struct {
		name       string
		driverID   int64
		actionType string
		payload    any

		getConnectionsByDriverCall getConnectionsByDriverCall
		postToConnectionCalls      []postToConnectionCall

		expectedErrMsg string
	}{
		{
			name:       "error fetching connections from store",
			driverID:   driverID,
			actionType: "test-action",
			payload:    "payload",
			getConnectionsByDriverCall: getConnectionsByDriverCall{
				driverID: driverID,
				err:      errors.New("dynamo error"),
			},
			expectedErrMsg: "dynamo error",
		},
		{
			name:       "no connections found succeeds with no pushes",
			driverID:   driverID,
			actionType: "test-action",
			payload:    "payload",
			getConnectionsByDriverCall: getConnectionsByDriverCall{
				driverID: driverID,
				result:   []store.WebSocketConnection{},
			},
		},
		{
			name:       "multiple connections all receive message",
			driverID:   driverID,
			actionType: "test-action",
			payload:    "payload",
			getConnectionsByDriverCall: getConnectionsByDriverCall{
				driverID: driverID,
				result: []store.WebSocketConnection{
					{DriverID: driverID, ConnectionID: "conn-1"},
					{DriverID: driverID, ConnectionID: "conn-2"},
					{DriverID: driverID, ConnectionID: "conn-3"},
				},
			},
			postToConnectionCalls: []postToConnectionCall{
				{connectionID: "conn-1"},
				{connectionID: "conn-2"},
				{connectionID: "conn-3"},
			},
		},
		{
			name:       "error on first push fails fast",
			driverID:   driverID,
			actionType: "test-action",
			payload:    "payload",
			getConnectionsByDriverCall: getConnectionsByDriverCall{
				driverID: driverID,
				result: []store.WebSocketConnection{
					{DriverID: driverID, ConnectionID: "conn-1"},
					{DriverID: driverID, ConnectionID: "conn-2"},
					{DriverID: driverID, ConnectionID: "conn-3"},
				},
			},
			postToConnectionCalls: []postToConnectionCall{
				{connectionID: "conn-1", err: errors.New("network failure")},
			},
			expectedErrMsg: "network failure",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := NewMockAPIGatewayManagementClient(t)
			mockConnLookup := NewMockConnectionLookup(t)

			mockConnLookup.EXPECT().GetConnectionsByDriver(mock.Anything, tc.getConnectionsByDriverCall.driverID).
				Return(tc.getConnectionsByDriverCall.result, tc.getConnectionsByDriverCall.err)

			expectedData := mustMarshal(t, Message{Action: tc.actionType, Payload: tc.payload})
			for _, call := range tc.postToConnectionCalls {
				mockClient.EXPECT().PostToConnection(mock.Anything, &apigatewaymanagementapi.PostToConnectionInput{
					ConnectionId: aws.String(call.connectionID),
					Data:         expectedData,
				}).Return(&apigatewaymanagementapi.PostToConnectionOutput{}, call.err)
			}

			pusher := NewPusher(mockClient, mockConnLookup)
			err := pusher.Broadcast(context.Background(), tc.driverID, tc.actionType, tc.payload)

			if tc.expectedErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}
