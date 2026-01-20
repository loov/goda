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
	sign := ""
	if size < 0 {
		sign = "-"
		size = -size
	}

	s := float64(size)

	switch {
	case s >= (1<<60)*2/3:
		return fmt.Sprintf("%s%.1fEB", sign, s/(1<<60))
	case s >= (1<<50)*2/3:
		return fmt.Sprintf("%s%.1fPB", sign, s/(1<<50))
	case s >= (1<<40)*2/3:
		return fmt.Sprintf("%s%.1fTB", sign, s/(1<<40))
	case s >= (1<<30)*2/3:
		return fmt.Sprintf("%s%.1fGB", sign, s/(1<<30))
	case s >= (1<<20)*2/3:
		return fmt.Sprintf("%s%.1fMB", sign, s/(1<<20))
	case s >= (1<<10)*2/3:
		return fmt.Sprintf("%s%.1fKB", sign, s/(1<<10))
	}

	return sign + strconv.Itoa(int(size)) + "B"
}

// ToStringShort returns a given size in bytes as a human size string.
// Examples: ToStringShort(1) ==> "1B"; ToStringShort(1000) ==> "1K"
func ToStringShort(size int64) string {
	sign := ""
	if size < 0 {
		sign = "-"
		size = -size
	}

	s := float64(size)

	switch {
	case s >= (1<<60)*2/3:
		return fmt.Sprintf("%s%.2fE", sign, s/(1<<60))
	case s >= (1<<50)*2/3:
		return fmt.Sprintf("%s%.2fP", sign, s/(1<<50))
	case s >= (1<<40)*2/3:
		return fmt.Sprintf("%s%.2fT", sign, s/(1<<40))
	case s >= (1<<30)*2/3:
		return fmt.Sprintf("%s%.2fG", sign, s/(1<<30))
	case s >= (1<<20)*2/3:
		return fmt.Sprintf("%s%.2fM", sign, s/(1<<20))
	case s >= (1<<10)*2/3:
		return fmt.Sprintf("%s%.2fK", sign, s/(1<<10))
	}

	return sign + strconv.Itoa(int(size)) + "B"
}
