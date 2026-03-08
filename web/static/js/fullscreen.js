(function () {

    const STORAGE_KEY = "lumenta.fullscreen";

    function updateClass(state) {
        document.documentElement.classList.toggle("is-fullscreen", state);
    }

    function turnOn() {
        updateClass(true);
        localStorage.setItem(STORAGE_KEY, "1");
    }

    function turnOff() {
        updateClass(false);
        localStorage.removeItem(STORAGE_KEY);
    }

    function init() {

        const onButton  = document.getElementById("fullscreen-on");
        const offButton = document.getElementById("fullscreen-off");

        if (!onButton) {
            turnOff();
            return;
        }

        const saved = localStorage.getItem(STORAGE_KEY) === "1";

        if (saved) {
            turnOn();
        } else {
            turnOff();
        }

        onButton.addEventListener("click", turnOn);

        if (offButton) {
            offButton.addEventListener("click", turnOff);
        }

        // ESC handler regisztráció
        Hotkeys.register("Escape", (e) => {

            if (!document.documentElement.classList.contains("is-fullscreen")) {
                return false;
            }

            const tag = document.activeElement?.tagName;

            if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") {
                return false;
            }

            turnOff();

            return true;

        }, 10);

    }

    document.addEventListener("DOMContentLoaded", init);

})();