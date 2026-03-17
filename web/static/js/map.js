(function () {

    const DEFAULT_PIN = `
    <svg width="28" height="40" viewBox="0 0 28 40"
        fill="currentColor"
        xmlns="http://www.w3.org/2000/svg">
    <path d="M14 39C14 39 26 26.8 26 15C26 7.8 20.6 2 14 2C7.4 2 2 7.8 2 15C2 26.8 14 39 14 39Z"
            stroke="white" stroke-width="2"/>
    </svg>
    `;

    function readPinSvg() {
        const node = document.getElementById("lumenta-map-pin");
        if (!node) return null;
        const svg = node.textContent.trim();
        return svg || null;
    }

    const pinSvg = readPinSvg() || DEFAULT_PIN;

    let container = null;

    let map = null;

    function initMap() {
        if (map) return;

        if (!container) return;

        const apiUrl = container.dataset.api;
        const enableCluster = container.dataset.cluster !== "false";
        const enablePopup = container.dataset.popup !== "false";
        const enableHover = container.dataset.hover !== "false";
        const maxPoints = parseInt(container.dataset.maxpoints || "100");

        if (window.MAP_POINT) {
            renderMap([window.MAP_POINT]);
        } else if (apiUrl) {
            fetch(apiUrl)
                .then(r => r.json())
                .then(data => renderMap(data.points || []));
        }

        function renderMap(points) {

            map = L.map(container, {
                zoomControl: true,
                preferCanvas: true
            });

            L.tileLayer(
                "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png",
                { attribution: "&copy; OpenStreetMap contributors" }
            ).addTo(map);

            if (!points.length) {
                map.setView([0, 0], 3);
                return;
            }

            if (points.length === 1) {
                addMarker(points[0]);
                map.setView([points[0].lat, points[0].lon], 16);
                return;
            }

            if (!enableCluster) {
                points.forEach(addMarker);
                fitBounds(points);
                return;
            }

            renderCluster(points);
            fitBounds(points);

            function fitBounds(points) {
                const bounds = L.latLngBounds(
                    points.map(p => [p.lat, p.lon])
                );
                map.fitBounds(bounds.pad(0.1), { maxZoom: 15 });
            }

            function renderCluster(points) {

                const features = points.map((p, i) => ({
                    type: "Feature",
                    properties: { index: i, ...p },
                    geometry: {
                        type: "Point",
                        coordinates: [p.lon, p.lat]
                    }
                }));

                const radius = Math.sqrt(
                    (window.innerWidth * window.innerHeight) / maxPoints
                );

                const index = new Supercluster({
                    radius: radius,
                    maxZoom: 18
                });

                index.load(features);

                const layer = L.layerGroup().addTo(map);

                function update() {
                    layer.clearLayers();

                    const b = map.getBounds();
                    const bbox = [b.getWest(), b.getSouth(), b.getEast(), b.getNorth()];
                    const zoom = map.getZoom();

                    const clusters = index.getClusters(bbox, zoom);

                    clusters.forEach(c => {
                        const [lng, lat] = c.geometry.coordinates;

                        if (c.properties.cluster) {
                            const count = c.properties.point_count;
                            const marker = L.marker([lat, lng], {
                                icon: clusterIcon(count)
                            }).addTo(layer);

                            marker.on("click", () => {
                                map.setView([lat, lng], zoom + 2);
                            });

                        } else {
                            addMarker(c.properties);
                        }
                    });
                }

                map.on("moveend zoomend", update);
                update();
            }

            function clusterIcon(count) {
                const color = clusterColor(count);

                return L.divIcon({
                    html: `<div class="lm-cluster" style="background:${color}">${count}</div>`,
                    className: "",
                    iconSize: [44, 44]
                });
            }

            function clusterColor(count) {
                if (count < 10) return "#7bc96f";
                if (count < 50) return "#41b6c4";
                if (count < 200) return "#2c7fb8";
                return "#253494";
            }

            function addMarker(p) {
                const marker = L.marker(
                    [p.lat, p.lon],
                    { icon: pinIcon(p.color) }
                ).addTo(map);

                if (enableHover && p.label) {
                    marker.bindTooltip(p.label);
                }

                if (enablePopup && (p.img || p.url || p.label)) {
                    marker.bindPopup(popupHtml(p));
                }
            }

            function pinIcon(color) {
                if (!color) {
                    color = "var(--icon-primary)";
                }
                return L.divIcon({
                    html: `<div style="color:${color}">${pinSvg}</div>`,
                    className: "",
                    iconSize: [28, 40],
                    iconAnchor: [14, 40]
                });
            }

            function popupHtml(p) {
                let html = `<div class="lm-popup">`;
                if (p.label) html += `<strong>${escape(p.label)}</strong>`;
                if (p.img) html += `<img src="${escape(p.img)}">`;
                html += `</div>`;

                if (p.url) {
                    return `<a href="${escape(p.url)}">${html}</a>`;
                }
                return html;
            }

            function escape(str) {
                return String(str)
                    .replaceAll("&", "&amp;")
                    .replaceAll("<", "&lt;")
                    .replaceAll(">", "&gt;")
                    .replaceAll("\"", "&quot;");
            }
        }
    }

    function ensureMap() {

        const rect = container.getBoundingClientRect();

        if (rect.width === 0 || rect.height === 0) {
            return;
        }

        if (!map) {
            initMap();
        } else {
            map.invalidateSize();
        }

    }

    //document.addEventListener("map:visible", resize);

    document.addEventListener("DOMContentLoaded",()=>{
        container = document.getElementById("map");
        if (!container) {
            return;
        }

        const ro = new ResizeObserver(() => {
            ensureMap();
        });
        ro.observe(container);

        ensureMap();
    
    });

})();
