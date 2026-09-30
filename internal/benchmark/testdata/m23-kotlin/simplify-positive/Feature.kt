class Feature {
    fun enabled(flag: Boolean): Boolean {
        return if (flag) {
            true
        } else {
            false
        }
    }
}
