function choose(state, other) {
  if (state.ready) {
    return 1;
  } else if (other) {
    return 2;
  } else if (state.ready) {
    return 3;
  }
  return 0;
}
