namespace Demo;
class Feature {
    int Classify(bool flag) {
        int Inner() {
            if (flag) {
                return 1;
            } else if (flag) {
                return 2;
            }
            return 3;
        }
        return Inner();
    }
}
