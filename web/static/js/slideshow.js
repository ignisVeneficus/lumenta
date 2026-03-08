(function () {
    const STORAGE_KEY = "lumenta.slideshow";
    const SLIDE_DELAY = window.SLIDESHOW?.delay ?? 5000;
    const UI_IDLE_DELAY = 2000;

    let slideTimer = null;
    let uiTimer = null;

    function nextUrl() {
        return window.SLIDESHOW.next;
    }

    function navigateNext() {
        const url = nextUrl();
        if (!url) {
            stop();
            return;
        }
        window.location.href = url;
    }

    function scheduleNext() {
        clearTimeout(slideTimer);
        slideTimer = setTimeout(navigateNext, SLIDE_DELAY);
    }
    /*
    function showUI() {
        document.documentElement.classList.remove("slideshow-ui-hidden");
        clearTimeout(uiTimer);
        uiTimer = setTimeout(() => {
            document.documentElement.classList.add("slideshow-ui-hidden");
        }, UI_IDLE_DELAY);
    }
    */
    function start() {
        localStorage.setItem(STORAGE_KEY, "1");
        scheduleNext();
        //showUI();
    }

    function stop() {
        const toggle = document.getElementById("toggle-slideshow");
        localStorage.removeItem(STORAGE_KEY);
        document.documentElement.classList.remove("slideshow-ui-hidden");
        clearTimeout(slideTimer);
        clearTimeout(uiTimer);
        if(toggle) toggle.checked = false;        
    }

    function init() {

        const toggle = document.getElementById("toggle-slideshow");
        const saved = localStorage.getItem(STORAGE_KEY) === "1";

        if (!toggle) {
            stop();
            return;
        }

        if (saved) {
            toggle.checked = true;
            start();
        }

        toggle.addEventListener("change", () => {
            if (toggle.checked) {
                start();
            } else {
                stop();
            }
        });
        /*
        document.addEventListener("mousemove", () => {
            if (!document.documentElement.classList.contains("slideshow-active")) {
                return;
            }
            showUI();

        });
        */

        Hotkeys.register("Escape", () => {
            if (!document.documentElement.classList.contains("slideshow-active")) {
                return false;
            }
            stop();
            return true;
        }, 20);
    }

    document.addEventListener("DOMContentLoaded", init);
})();