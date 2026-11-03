int classify(bool flag) {
  int inner() {
    if (flag) {
      return 1;
    } else if (flag) {
      return 2;
    }
    return 3;
  }
  return inner();
}
