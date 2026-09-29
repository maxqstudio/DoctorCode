function choose(flag, state) {
  if (flag) {
    return 1;
  } else if (state.ready) {
    return 2;
  } else if (flag) {
    return 3;
  }
  return 0;
}
