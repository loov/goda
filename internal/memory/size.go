package memory

import (
	"fmt"
	"strconv"
)

type Bytes int64

func (bytes Bytes) String() string {
	return ToString(int64(bytes))
}

// ToString returns a given size in bytes as a human size string.
// Examples: ToString(1) ==> "1B"; ToString(1000) ==> "1KB"
func ToString(size int64) string {
	s := float64(size)

	switch {
	case s >= (1<<60)*2/3:
		return fmt.Sprintf("%.1fEB", s/(1<<60))
	case s >= (1<<50)*2/3:
		return fmt.Sprintf("%.1fPB", s/(1<<50))
	case s >= (1<<40)*2/3:
		return fmt.Sprintf("%.1fTB", s/(1<<40))
	case s >= (1<<30)*2/3:
		return fmt.Sprintf("%.1fGB", s/(1<<30))
	case s >= (1<<20)*2/3:
		return fmt.Sprintf("%.1fMB", s/(1<<20))
	case s >= (1<<10)*2/3:
		return fmt.Sprintf("%.1fKB", s/(1<<10))
	}

	return strconv.Itoa(int(size)) + "B"
}
