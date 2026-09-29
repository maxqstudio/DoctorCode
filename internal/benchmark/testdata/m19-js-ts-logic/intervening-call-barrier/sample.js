function choose(flag) {
  if (flag) {
    return 1;
  } else if (mutate()) {
    return 2;
  } else if (flag) {
    return 3;
  }
  return 0;
}
