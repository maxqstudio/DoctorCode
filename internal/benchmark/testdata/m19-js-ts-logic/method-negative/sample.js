class Example {
  choose(flag, other) {
    if (flag) {
      return 1;
    } else if (other) {
      return 2;
    } else if (flag) {
      return 3;
    }
    return 0;
  }
}
