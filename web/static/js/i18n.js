window.t = function(key, params) {
  let str = window.I18N[key];
  if (!str) return `[${key}]`;

  if (params) {
    for (const k in params) {
      str = str.replaceAll(`{${k}}`, params[k]);
    }
  }

  return str;
};