package corpus

func side() bool { return true }

func Route() int {
	if side() {
		return 1
	} else if side() {
		return 2
	}
	return 0
}
