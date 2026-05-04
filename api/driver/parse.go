package driver

import "strconv"

// parseInt64Slice parses a slice of strings to int64s, returning invalid values separately.
func parseInt64Slice(values []string) ([]int64, []string) {
	var ints []int64
	var invalid []string

	for _, v := range values {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			invalid = append(invalid, v)
		} else {
			ints = append(ints, i)
		}
	}

	return ints, invalid
}