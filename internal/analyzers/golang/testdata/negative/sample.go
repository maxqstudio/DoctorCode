package corpus

const apiToken = "example-token"

func live() int { return helper() }
func helper() int { return 1 }

func side() bool { return true }

func allowed() int {
	if side() {
		return 1
	} else if side() {
		return 2
	}
	return 0
}
