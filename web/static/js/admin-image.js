(function () {
  const elCovers = document.getElementById("covers");
  if (!elCovers) return;

  function success(response){
    console.debug("success",window.showFlash);
    if (window.showFlash){
      window.showFlash({
        type: "success",
        title: "Saved",
        description: "Changes saved"
      });
    }
  }

  function error(response){
    if (window.showFlash){
      window.showFlash({
        type: "error",
        title: "Error",
        description: "Changes not saved"
      });
    }
  }


  function sendPatch(path, payload) {
    return fetch(path, {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
      credentials: "same-origin",
    })
    .then(response => {
      if (!response.ok) {
        throw response;
      }
      success(response);
      return response;
    })
    .catch(err => {
      error(err);
      throw err;
    });
  }

  function handleCovers(event, isSelect) {
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
      error(err);
    });
  }

  function handleACL(event) {
    const selected= parseInt(event.target.value);
    if(selected == NaN) return
    const path = window.ROUTES.image
    var userID = 0;
    if (selected == 1){
      userID = 0;
    }
    const payload = {
      "acl_scope":selected,
      "acl_user_id":userID
    }
    sendPatch(path,payload)
      .then().catch((err) => {
      error(err);
    });
  }

  function handleFocus(event) {
    const selected= event.target.value;
    const path = window.ROUTES.image

    const payload = {
      "focus":selected,
    }
    if (selected== "manual"){
      payload["focus_x"]=0.5;
      payload["focus_y"]=0.5;
    }
    sendPatch(path,payload)
      .then().catch((err) => {
      error(err);
    });
  }


  elCovers.addEventListener("xselect:select", (e) => handleCovers(e, true));
  elCovers.addEventListener("xselect:deselect", (e) => handleCovers(e, false));

  document.querySelectorAll('input[name="acl_level"]').forEach(radio => {
      radio.addEventListener("change", (e) =>handleACL(e));
  });

  document.querySelectorAll('input[name="focus"]').forEach(radio => {
      radio.addEventListener("change", (e) =>handleFocus(e));
  });

})();