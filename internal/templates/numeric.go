package templates

import (
	"math"
	"strconv"
	"text/template"

	"github.com/loov/goda/internal/memory"
)

func numericFuncs() template.FuncMap {
	return template.FuncMap{
		"add": add,
		"div": div,
		"sub": sub,
		"mul": mul,

		"float": toFloat64,
		"int":   func(v interface{}) int64 { return int64(toFloat64(v)) },
		"round": func(v interface{}) float64 { return math.Round(toFloat64(v)) },

		"log":   func(v interface{}) float64 { return math.Log(toFloat64(v)) },
		"log10": func(v interface{}) float64 { return math.Log10(toFloat64(v)) },
		"log2":  func(v interface{}) float64 { return math.Log2(toFloat64(v)) },
	}
}

func add(xs ...interface{}) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	total := toFloat64(xs[0])
	for _, x := range xs[1:] {
		total += toFloat64(x)
	}
	return total
}

func div(xs ...interface{}) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	total := toFloat64(xs[0])
	for _, x := range xs[1:] {
		total /= toFloat64(x)
	}
	return total
}

func sub(xs ...interface{}) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	total := toFloat64(xs[0])
	for _, x := range xs[1:] {
		total -= toFloat64(x)
	}
	return total
}

func mul(xs ...interface{}) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	total := toFloat64(xs[0])
	for _, x := range xs[1:] {
		total *= toFloat64(x)
	}
	return total
}

func toFloat64(v interface{}) float64 {
	switch v := v.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case int16:
		return float64(v)
	case int8:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint16:
		return float64(v)
	case uint8:
		return float64(v)
	case memory.Bytes:
		return float64(v)
	case string:
		if x, err := strconv.ParseFloat(v, 64); err == nil {
			return x
		}
		return math.NaN()
	case bool:
		if v {
			return 1
		}
		return 0
	default:
		return math.NaN()
	}
}
