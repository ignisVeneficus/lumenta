function enableEventuallyAvailableImage(img, {
  maxRetries = 20,
  initialDelay = 1000,
  maxDelay = 15000,
} = {}) {

  if (img._retryBound) return;
  img._retryBound = true;

  let attempt = 0;
  let delay = initialDelay;
  let timer = null;

  function scheduleRetry() {
    if (timer) return;
    if (attempt >= maxRetries) {
      console.warn("Image retry exhausted:", img.src);
      retrying = false;
      return;
    }

    attempt++;

    timer = setTimeout(() => {
      timer = null;

      const url = new URL(img.src, location.href);
      url.searchParams.set("_r", Date.now());

      console.info("retry attempt:", attempt, url.toString());

      img.src = url.toString();

      delay = Math.min(delay * 1.7, maxDelay);

      scheduleRetry();

    }, delay);
  }
  let retrying = false;

  img.addEventListener("error", () => {
    if (!retrying) {
      retrying = true;
      scheduleRetry();
    }
  });

  img.addEventListener("load", () => {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    console.log("Loaded:", img.src);
  });

  if (img.complete) {
    if (img.naturalWidth > 0) {
      console.log("Already loaded", img.src);
    } else {
      console.log("Already failed → retry", img.src);
      scheduleRetry();
    }
  }
}
function initDerivativeRetry(root = document) {
  root.querySelectorAll(".derivative-img").forEach(img => {
    enableEventuallyAvailableImage(img);
  });
}

document.addEventListener("DOMContentLoaded", () => {
  initDerivativeRetry();
});