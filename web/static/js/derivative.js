function enableEventuallyAvailableImage(img, {
  maxRetries = 20,
  initialDelay = 1000,
  maxDelay = 15000,
} = {}) {
  console.info("start:", img.src);
  let attempt = 0;
  let delay = initialDelay;
  let timer = null;

  function retry() {
    console.info("retry", img.src);
    if (attempt >= maxRetries) {
      console.warn("Image retry exhausted:",  img.src);
      return;
    }

    attempt++;

    timer = setTimeout(() => {
      timer = null;
      const ts= Date.now();
      
      const base =  img.src;
      const url = new URL(base, location.href);
      url.searchParams.set("_r", Date.now());
      console.info("attempt:",attempt,base);

      img.src = url.toString();

      delay = Math.min(delay * 1.7, maxDelay);

    }, delay);
  }

  img.addEventListener("error", () => {
    retry();
  });

  img.addEventListener("load", () => {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    console.log("Derivative loaded:", img.src);
  });

  if (img.complete && img.naturalWidth > 0) {
    console.log("Already loaded",img.src);
  }
}

document
  .querySelectorAll(".derivative-img")
  .forEach(img => enableEventuallyAvailableImage(img));