package corpus

func Entry() int {
	fn := helper
	return fn()
}

func helper() int { return 1 }
