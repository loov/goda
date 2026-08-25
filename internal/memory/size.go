package memory

import (
	"fmt"
	"math"
	"strconv"
)

type Bytes int64

func (bytes Bytes) String() string {
	return ToString(int64(bytes))
}

// ToString returns a given size in bytes as a human size string.
// Examples: ToString(1) ==> "1B"; ToString(1000) ==> "1KB"
func ToString(size int64) string {
	return format(size, "%s%.1f%sB")
}

// ToStringShort returns a given size in bytes as a human size string.
// Examples: ToStringShort(1) ==> "1B"; ToStringShort(1000) ==> "1K"
func ToStringShort(size int64) string {
	return format(size, "%s%.2f%s")
}

var units = []struct {
	shift  uint
	suffix string
}{{60, "E"}, {50, "P"}, {40, "T"}, {30, "G"}, {20, "M"}, {10, "K"}}

// format renders size with the largest unit where the value is at least 2/3.
func format(size int64, layout string) string {
	sign := ""
	if size < 0 {
		sign = "-"
	}
	s := math.Abs(float64(size))
	for _, u := range units {
		if s >= float64(uint64(1)<<u.shift)*2/3 {
			return fmt.Sprintf(layout, sign, s/float64(uint64(1)<<u.shift), u.suffix)
		}
	}
	return sign + strconv.FormatInt(int64(s), 10) + "B"
}
