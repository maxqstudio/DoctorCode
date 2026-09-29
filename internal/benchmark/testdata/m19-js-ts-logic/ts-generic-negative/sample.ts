function choose<T>(flag: T | null, other: boolean): number {
  if (flag === null) {
    return 1;
  } else if (other) {
    return 2;
  } else if (flag === null) {
    return 3;
  }
  return 0;
}
