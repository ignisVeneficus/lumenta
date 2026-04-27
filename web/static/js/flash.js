(function () {
  const container = document.getElementById("flash-container");
  if (!container) return;

  function renderFlash({ type = "success", title = "", description = "" }) {
    const el = document.createElement("div");
    el.className = `flash flash-${type}`;

    if (title) {
      const t = document.createElement("div");
      t.className = "flash-title";
      t.textContent = title;
      el.appendChild(t);
    }

    if (description) {
      const d = document.createElement("div");
      d.className = "flash-description";
      d.textContent = description;
      el.appendChild(d);
    }

    container.appendChild(el);

    // auto remove (animation után)
    setTimeout(() => el.remove(), 3000);
  }

  // 🔹 server flash
  if (window.FLASH_FROM_SERVER) {
    renderFlash(window.FLASH_FROM_SERVER);
  }

  // 🔹 global API
  window.showFlash = renderFlash;
})();