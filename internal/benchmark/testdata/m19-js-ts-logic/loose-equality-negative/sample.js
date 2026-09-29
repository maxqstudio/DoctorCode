function choose(flag, other) {
  if (flag == true) {
    return 1;
  } else if (other) {
    return 2;
  } else if (flag == true) {
    return 3;
  }
  return 0;
}
