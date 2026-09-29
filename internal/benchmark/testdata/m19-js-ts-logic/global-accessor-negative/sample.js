Object.defineProperty(globalThis, "flag", {
  get() {
    return Math.random() > 0.5;
  },
});

function choose(other) {
  if (flag) {
    return 1;
  } else if (other) {
    return 2;
  } else if (flag) {
    return 3;
  }
  return 0;
}
