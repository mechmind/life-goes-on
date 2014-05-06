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

func ibound(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high-1 {
		return high - 1
	}

	return value
}
