package statistics

import "errors"

// Operation-context sentinels wrapped around repository failures so callers
// can identify which aggregation step failed.
var (
	errCountEndpoints           = errors.New("count endpoints")
	errBaseStationStats         = errors.New("get base station stats")
	errMessageStats             = errors.New("get message stats")
	errTimeSeries               = errors.New("get time series")
	errEndpointMessageCounts    = errors.New("get endpoint message counts")
	errBaseStationMessageCounts = errors.New("get base station message counts")
	errParseActivityDay         = errors.New("parse activity day")
)
