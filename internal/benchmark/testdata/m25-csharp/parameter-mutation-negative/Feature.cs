namespace Demo;
class Feature {
    int Classify(bool flag) {
        flag = !flag;
        if (flag) {
            return 1;
        } else if (flag) {
            return 2;
        }
        return 3;
    }
}
