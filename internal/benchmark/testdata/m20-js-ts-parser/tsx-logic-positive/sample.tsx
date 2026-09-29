export function choose(flag: boolean, other: boolean): number {
  if (flag) {
    return 1;
  } else if (other) {
    return 2;
  } else if (flag) {
    return 3;
  }
  return 0;
}
const node = <span>{choose(true, false)}</span>;
