function decision(flag) {
  return flag;
}

const apiToken = process.env.API_TOKEN;
console.log(decision(true), apiToken);
