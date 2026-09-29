function choose(flag: boolean | null): number {
  if (flag === null) {
    return 1;
  } else if (flag === true) {
    return 2;
  } else if (flag === null) {
    return 3;
  }
  return 0;
}
