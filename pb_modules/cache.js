// Shared cache module
const cache = {};
const expirations = {};

function set(key, value, ttl) {
  cache[key] = value;
  
  if (ttl) {
    expirations[key] = Date.now() + (ttl * 1000);
    
    setTimeout(() => {
      if (expirations[key] && Date.now() >= expirations[key]) {
        delete cache[key];
        delete expirations[key];
      }
    }, ttl * 1000);
  }
  
  return true;
}

function get(key) {
  if (expirations[key] && Date.now() >= expirations[key]) {
    delete cache[key];
    delete expirations[key];
    return null;
  }
  
  return cache[key];
}

function has(key) {
  return get(key) !== undefined && get(key) !== null;
}

function del(key) {
  delete cache[key];
  delete expirations[key];
  return true;
}

function clear() {
  Object.keys(cache).forEach(k => delete cache[k]);
  Object.keys(expirations).forEach(k => delete expirations[k]);
}

function keys() {
  return Object.keys(cache);
}

function size() {
  return Object.keys(cache).length;
}

exports.set = set;
exports.get = get;
exports.has = has;
exports.del = del;
exports.clear = clear;
exports.keys = keys;
exports.size = size;
