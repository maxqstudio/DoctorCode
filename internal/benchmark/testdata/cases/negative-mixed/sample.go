package corpus

const apiToken = "example-token"

func Entry(v int) int { return helper(v) + second(v) }
func helper(v int) int { return v }
func second(v int) int { return v }

func side() bool { return true }

func Allowed() int {
	if side() {
		return 1
	} else if side() {
		return 2
	}
	return 0
}
