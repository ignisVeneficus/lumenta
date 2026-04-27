document.addEventListener("DOMContentLoaded", () => {
  document.querySelectorAll(".clickable-row").forEach(row => {
    row.addEventListener("click", (e) => {
      if (e.target.closest("a, button")) return;

      const href = row.dataset.href;
      if (href) window.location.href = href;
    });
  });
});