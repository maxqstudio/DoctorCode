package corpus

func Entry(v int) int { return wrapper(v) }
func target(v int) int { return v }
func wrapper(v int) int { return target(v) }

func Noise() int {
	wrapper := 1
	return wrapper
}
