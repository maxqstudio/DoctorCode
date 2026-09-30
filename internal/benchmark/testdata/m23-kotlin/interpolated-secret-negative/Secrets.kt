class Secrets {
    private val suffix = System.getenv("TOKEN_SUFFIX")
    private val apiToken = "hardcoded-$suffix"
}
