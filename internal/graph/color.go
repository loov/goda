package graph

import (
	"fmt"
	"math"
)

func hslahex(h, s, l, a float64) string {
	r, g, b, xa := hsla(h, s, l, a)
	return fmt.Sprintf("\"#%02x%02x%02x%02x\"", sat8(r), sat8(g), sat8(b), sat8(xa))
}

func hue(v1, v2, h float64) float64 {
	if h < 0 {
		h += 1
	}
	if h > 1 {
		h -= 1
	}
	if 6*h < 1 {
		return v1 + (v2-v1)*6*h
	} else if 2*h < 1 {
		return v2
	} else if 3*h < 2 {
		return v1 + (v2-v1)*(2.0/3.0-h)*6
	}

	return v1
}

func hsla(h, s, l, a float64) (r, g, b, ra float64) {
	if s == 0 {
		return l, l, l, a
	}

	_, h = math.Modf(h)

	var v2 float64
	if l < 0.5 {
		v2 = l * (1 + s)
	} else {
		v2 = (l + s) - s*l
	}

	v1 := 2*l - v2
	r = hue(v1, v2, h+1.0/3.0)
	g = hue(v1, v2, h)
	b = hue(v1, v2, h-1.0/3.0)
	ra = a

	return
}

// sat8 converts 0..1 float to 0..255 uint8.
// sat8 is short for saturate 8, referring to 8 byte saturation arithmetic.
//
//     sat8(x) = 0   if x < 0
//     sat8(x) = 255 if x > 1
func sat8(v float64) uint8 {
	v *= 255.0
	if v >= 255 {
		return 255
	} else if v <= 0 {
		return 0
	}
	return uint8(v)
}
