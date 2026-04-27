(function () {
  const el = document.getElementById("covers");
  if (!el) return;

  function sendPatch(path, payload) {
    return fetch(path, {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
      credentials: "same-origin", 
    });
  }

  function handle(event, isSelect) {
    if (!event.detail || !event.detail.node || !event.detail.node.id) return;

    const albumId = event.detail.node.id;

    // URL build
    const path = buildPath(window.ROUTES.album, {
      "id": albumId,
    });

    // payload
    const payload = isSelect
      ? { cover_image_id: window.IMAGE_ID }
      : { cover_image_id: null }; 

    // request
    sendPatch(path, payload).catch((err) => {
      console.error("PATCH failed:", err);
    });
  }

  el.addEventListener("xselect:select", (e) => handle(e, true));
  el.addEventListener("xselect:deselect", (e) => handle(e, false));
})();