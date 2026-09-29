function decision(flag) {
  if (flag) {
    return true;
  } else {
    return false;
  }
}
const view = <div>{String(decision(true))}</div>;
