function decision(flag: boolean): boolean {
  if (flag) {
    return true;
  } else {
    return false;
  }
}

const apiToken: string = "hardcoded-production-token-12345";
console.log(decision(true), apiToken);
