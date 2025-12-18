package iracing

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func loadFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestClient_GetUserInfo(t *testing.T) {
	testCases := []struct {
		name               string
		linkResponseFile   string
		linkStatusCode     int
		memberResponseFile string
		memberStatusCode   int
		expectedUserID     int64
		expectedUserName   string
		expectedMemberYear int
		expectedErr        string
	}{
		{
			name:               "success",
			linkResponseFile:   "fixtures/member/info_link_response.json",
			linkStatusCode:     http.StatusOK,
			memberResponseFile: "fixtures/member/info_response.json",
			memberStatusCode:   http.StatusOK,
			expectedUserID:     1100750,
			expectedUserName:   "Jon Sabados",
			expectedMemberYear: 2024,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			linkResponse := loadFixture(t, tc.linkResponseFile)
			memberResponse := loadFixture(t, tc.memberResponseFile)

			httpClient := NewMockHTTPClient(t)
			metricsClient := NewMockMetricsClient(t)

			// First call: get the link
			httpClient.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.String() == "https://test.iracing.com/data/member/info"
			})).Return(&http.Response{
				StatusCode: tc.linkStatusCode,
				Body:       io.NopCloser(strings.NewReader(linkResponse)),
			}, nil)

			// Second call: fetch from S3
			httpClient.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "scorpio-assets.s3")
			})).Return(&http.Response{
				StatusCode: tc.memberStatusCode,
				Body:       io.NopCloser(strings.NewReader(memberResponse)),
			}, nil)

			client := NewClient(httpClient, metricsClient, WithBaseURL("https://test.iracing.com"))

			userInfo, err := client.GetUserInfo(context.Background(), "test-access-token")

			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, userInfo)
				assert.Equal(t, tc.expectedUserID, userInfo.UserID)
				assert.Equal(t, tc.expectedUserName, userInfo.UserName)
				assert.Equal(t, tc.expectedMemberYear, userInfo.MemberSince.Year())
			}
		})
	}
}
