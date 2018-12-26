package memory

import (
	"fmt"
	"strconv"
)

type Bytes int64

func (bytes Bytes) String() string {
	return ToString(int64(bytes))
}

func ToString(s int64) string {
	size := float64(s)

	switch {
	case size >= (1<<60)*2/3:
		return fmt.Sprintf("%.1fEB", size/(1<<60))
	case size >= (1<<50)*2/3:
		return fmt.Sprintf("%.1fPB", size/(1<<50))
	case size >= (1<<40)*2/3:
		return fmt.Sprintf("%.1fTB", size/(1<<40))
	case size >= (1<<30)*2/3:
		return fmt.Sprintf("%.1fGB", size/(1<<30))
	case size >= (1<<20)*2/3:
		return fmt.Sprintf("%.1fMB", size/(1<<20))
	case size >= (1<<10)*2/3:
		return fmt.Sprintf("%.1fKB", size/(1<<10))
	}

	return strconv.Itoa(int(s)) + "B"
}
