function decision(flag) {
  if (flag) {
    return true;
  } else {
    return false;
  }
}

const apiToken = "hardcoded-production-token-12345";
console.log(decision(true), apiToken);
