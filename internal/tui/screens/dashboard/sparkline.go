package dashboard

// blockRunes are the eight vertical block glyphs used to draw sparklines and
// mini bars, ordered from shortest (▁) to tallest (█).
var blockRunes = []rune("▁▂▃▄▅▆▇█")

// sparkline renders values as a single row of block runes exactly width columns
// wide. When there are more values than columns they are bucketed (averaged)
// down; when there are fewer, each value maps to one column and the line is
// shorter than width. Values are normalized across their own min..max range so
// the tallest sample is █ and the shortest is ▁.
//
// Edge cases: an empty slice or non-positive width yields "". A slice whose
// values are all equal (including a single value) renders as the lowest rune so
// the line reads as a flat baseline rather than a spike.
func sparkline(values []float64, width int) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}

	cols := resample(values, width)

	min, max := cols[0], cols[0]
	for _, v := range cols {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	span := max - min
	out := make([]rune, len(cols))
	for i, v := range cols {
		out[i] = runeForFraction(fraction(v, min, span))
	}
	return string(out)
}

// sparklineInt is a convenience wrapper for integer series (the common case for
// PiHole history counts).
func sparklineInt(values []int, width int) string {
	fs := make([]float64, len(values))
	for i, v := range values {
		fs[i] = float64(v)
	}
	return sparkline(fs, width)
}

// fraction returns v's position within [min, min+span] as a 0..1 value,
// guarding
// against a zero span (flat series) by returning 0.
func fraction(v, min, span float64) float64 {
	if span <= 0 {
		return 0
	}
	return (v - min) / span
}

// runeForFraction maps a 0..1 fraction to one of the block runes.
func runeForFraction(f float64) rune {
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	idx := int(f*float64(len(blockRunes)-1) + 0.5)
	return blockRunes[idx]
}

// resample reduces (or passes through) values to at most width buckets. When
// len(values) <= width the slice is returned as-is. Otherwise consecutive
// values are grouped into width contiguous buckets and each bucket is averaged.
func resample(values []float64, width int) []float64 {
	n := len(values)
	if n <= width {
		return values
	}

	out := make([]float64, width)
	for i := 0; i < width; i++ {
		start := i * n / width
		end := (i + 1) * n / width
		if end <= start {
			end = start + 1
		}
		if end > n {
			end = n
		}
		var sum float64
		for j := start; j < end; j++ {
			sum += values[j]
		}
		out[i] = sum / float64(end-start)
	}
	return out
}
