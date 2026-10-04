class Feature {
    fun classify(flag: Boolean): Int {
        val supplier = {
            if (flag) {
                true
            } else if (flag) {
                false
            } else {
                false
            }
        }
        return if (supplier()) 1 else 0
    }
}
