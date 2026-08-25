package memory
import "testing"
func TestFormat(t *testing.T) {
	for _, c := range []struct{ v int64; l, s string }{
		{1, "1B", "1B"}, {1000, "1.0KB", "0.98K"}, {-1500000, "-1.4MB", "-1.43M"}, {1 << 40, "1.0TB", "1.00T"}, {0, "0B", "0B"},
	} {
		if g := ToString(c.v); g != c.l { t.Errorf("%d: %q != %q", c.v, g, c.l) }
		if g := ToStringShort(c.v); g != c.s { t.Errorf("%d: %q != %q", c.v, g, c.s) }
	}
}
