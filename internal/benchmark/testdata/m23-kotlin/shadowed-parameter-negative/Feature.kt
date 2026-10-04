class Feature {
    fun classify(flag: Boolean): Int {
        val flag = !flag
        if (flag) {
            return 1
        } else if (flag) {
            return 2
        }
        return 3
    }
}
