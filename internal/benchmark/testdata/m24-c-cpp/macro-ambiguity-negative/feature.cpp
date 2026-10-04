#define CHECK(x) (x)
int classify(bool flag) {
    if (CHECK(flag)) { return 1; }
    if (CHECK(flag)) { return 2; }
    return 0;
}
