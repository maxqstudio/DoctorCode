function choose(flag, other) {
  if (true === flag) {
    return 1;
  } else if (other) {
    return 2;
  } else if (true === flag) {
    return 3;
  }
  return 0;
}
