function choose(flag, other) {
  if (flag /* stable parameter */) {
    return 1;
  } else /* chain */ if (other) {
    return 2;
  } else /* duplicate */ if (flag) {
    return 3;
  }
  return 0;
}
