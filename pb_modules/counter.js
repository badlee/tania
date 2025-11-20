// Shared counter module
let i = 0;
const history = [];

function inc() {
  i++;
  history.push({ value: i, timestamp: Date.now() });
  return i;
}

function dec() {
  i--;
  history.push({ value: i, timestamp: Date.now() });
  return i;
}

function get() {
  return i;
}

function reset() {
  const old = i;
  i = 0;
  history.push({ value: i, timestamp: Date.now(), reset: true });
  return old;
}

function getHistory() {
  return history;
}

exports.inc = inc;
exports.dec = dec;
exports.get = get;
exports.reset = reset;
exports.getHistory = getHistory;
