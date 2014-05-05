package main

func fabs(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

func fbound(value, low, high float32) float32 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}

	return value
}
