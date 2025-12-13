package iracing

import "time"

// LapTimeToDuration can be used to convert iRacing lap times, which are in 10ths of milliseconds (e.g., 1234 = 123.4ms), to go durations.
func LapTimeToDuration(duration int) time.Duration {
	return time.Duration(duration) * 100 * time.Microsecond
}
