function choose(flag, other) {
  if (flag !== false) {
    return 1;
  } else if (other) {
    return 2;
  } else if (flag !== false) {
    return 3;
  }
  return 0;
}
