(function () {
    const handlers = new Map(); // key -> [{handler, priority}]
    window.Hotkeys = {
        register(key, handler, priority = 0) {
            if (!handlers.has(key)) {
                handlers.set(key, []);
            }
            const list = handlers.get(key);
            const entry = { handler, priority };
            list.push(entry);
            list.sort((a, b) => b.priority - a.priority);
            return entry;
        },
        unregister(entry) {
            for (const list of handlers.values()) {
                const i = list.indexOf(entry);
                if (i >= 0) {
                    list.splice(i, 1);
                    return;
                }
            }
        }
    };

    document.addEventListener("keydown", (e) => {
        const list = handlers.get(e.key);
        if (!list) return;
        
        for (const h of list) {
            try {
                const handled = h.handler(e);
                if (handled) {
                    e.preventDefault();
                    e.stopPropagation();
                    return;
                }
            } catch (err) {
                console.error("Hotkey handler error", err);
            }
        }
    });
})();