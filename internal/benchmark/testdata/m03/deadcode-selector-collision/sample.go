package corpus

type holder struct{}

func (holder) unusedHelper() {}

func Entry(h holder) {
	h.unusedHelper()
}

func unusedHelper() {}
