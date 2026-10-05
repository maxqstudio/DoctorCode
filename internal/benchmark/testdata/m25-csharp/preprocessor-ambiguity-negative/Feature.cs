namespace Demo;
class Feature {
    int Classify(bool flag) {
#if FEATURE_A
        if (flag) {
            return 1;
        } else if (flag) {
            return 2;
        }
#endif
        return 3;
    }
}
