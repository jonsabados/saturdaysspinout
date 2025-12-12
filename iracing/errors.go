package iracing

import "errors"

// ErrUpstreamUnauthorized is returned when iRacing returns 401, indicating the access token is expired
var ErrUpstreamUnauthorized = errors.New("upstream returned 401 unauthorized")
