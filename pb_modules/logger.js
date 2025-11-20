// Shared logger module
const logs = [];

function info(message, ...args) {
  const entry = {
    level: "INFO",
    message: message,
    args: args,
    timestamp: new Date().toISOString()
  };
  logs.push(entry);
  log("[INFO]", message, ...args);
}

function warn(message, ...args) {
  const entry = {
    level: "WARN",
    message: message,
    args: args,
    timestamp: new Date().toISOString()
  };
  logs.push(entry);
  log("[WARN]", message, ...args);
}

function error(message, ...args) {
  const entry = {
    level: "ERROR",
    message: message,
    args: args,
    timestamp: new Date().toISOString()
  };
  logs.push(entry);
  log("[ERROR]", message, ...args);
}

function getLogs(level) {
  if (level) {
    return logs.filter(l => l.level === level);
  }
  return logs;
}

function clear() {
  logs.length = 0;
}

exports.info = info;
exports.warn = warn;
exports.error = error;
exports.getLogs = getLogs;
exports.clear = clear;
