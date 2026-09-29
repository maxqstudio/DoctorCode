function choose() {
  if (ready()) {
    return 1;
  } else if (other()) {
    return 2;
  } else if (ready()) {
    return 3;
  }
  return 0;
}
