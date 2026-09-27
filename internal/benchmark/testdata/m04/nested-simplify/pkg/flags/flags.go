package flags

func Normalize(ok bool) bool {
	if ok {
		return true
	} else {
		return false
	}
}
