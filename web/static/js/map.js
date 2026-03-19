(function () {

    const DEFAULT_PIN = `
    <svg width="28" height="40" viewBox="0 0 28 40"
        fill="currentColor"
        xmlns="http://www.w3.org/2000/svg">
    <path d="M14 39C14 39 26 26.8 26 15C26 7.8 20.6 2 14 2C7.4 2 2 7.8 2 15C2 26.8 14 39 14 39Z"
            stroke-width="2"/>
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
    let mapInitializing = false;

    function initMap() {
        if (map || mapInitializing) return;

        if (!container) return;

        mapInitializing = true;

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
                .then(resp => renderMap(resp.data || []));
        }else{
            console.error("no api url!")
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
                mapInitializing = false;
                return;
            }

            if (points.length === 1) {
                addMarker(null,points[0]);
                map.setView([points[0].lat, points[0].lon], 16);
                mapInitializing = false;
                return;
            }

            if (!enableCluster) {
                points.forEach(null,addMarker);
                fitBounds(points);
                mapInitializing = false;
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
                                const clusterId = c.properties.cluster_id;
                                const expansionZoom = index.getClusterExpansionZoom(clusterId);
                                const currentZoom = map.getZoom();
                                if (currentZoom >= expansionZoom || currentZoom >= 18) {
                                    const leaves = index.getLeaves(clusterId, 50); // limit!
                                    layer.clearLayers();
                                    const spiderPoints = spiderfyPoints(
                                        map,
                                        { lat, lon: lng },
                                        leaves.map(l => l.properties)
                                    );

                                    spiderPoints.forEach(p => addMarker(layer, p));

                                } else {
                                    map.setView([lat, lng], expansionZoom);
                                }
                            });
                        } else {
                            addMarker(layer, c.properties);
                        }
                    });
                }

                map.on("moveend zoomend", update);
            }

            function clusterIcon(count) {
                const level = Math.floor(Math.log10(count));
                const bucket = clusterBucket(count)

                return L.divIcon({
                    html: `<div class="lm-cluster lm-cluster-${level} lm-cluster-points-${bucket}">${count}</div>`,
                    className: "",
                    iconSize: [44, 44]
                });
            }

            function clusterBucket(count) {
                if (count <= 1) return 1;

                const exp = Math.floor(Math.log10(count));
                const base = Math.pow(10, exp);
                const normalized = count / base;

                let bucket;
                if (normalized <= 1) bucket = 1;
                else if (normalized <= 2) bucket = 2;
                else if (normalized <= 5) bucket = 5;
                else bucket = 10;

                return bucket * base;
            }

            function addMarker(layer, p) {
                place = layer;
                if(place==null){
                    place = map;
                }
                const marker = L.marker(
                    [p.lat, p.lon],
                    { icon: pinIcon(p.color) }
                ).addTo(place);

                if (enableHover && p.label) {
                    marker.bindTooltip(p.label);
                }

                if (enablePopup && (p.img || p.url || p.label)) {
                    marker.bindPopup(popupHtml(p));
                }
            }

            function pinIcon(color) {
                if (!color) {
                    color = "primary";
                }
                return L.divIcon({
                    html: `<div class="pin pin-${color}">${pinSvg}</div>`,
                    className: "",
                    iconSize: [28, 40],
                    iconAnchor: [14, 40]
                });
            }

            function popupHtml(p) {
                let html = `<div class="lm-popup">`;
                if (p.label) html += `<div class="label">${escape(p.label)}</div>`;
                if (p.img) html += `<div class="img-wrapper"><img src="${escape(p.imgUrl)}"></div>`;
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

    function spiderfyPoints(map, center, points) {
        const centerPt = map.project([center.lat, center.lon]);
        const res = [];

        const separation = 25; // px
        const spiralLengthFactor = 5;
        const spiralFootSeparation = 28;

        let angle = 0;

        for (let i = 0; i < points.length; i++) {

            const r = spiralLengthFactor * angle + 20;

            const x = centerPt.x + r * Math.cos(angle);
            const y = centerPt.y + r * Math.sin(angle);

            const latlng = map.unproject([x, y]);

            res.push({
                ...points[i],
                _spiderfy: true,
                lat: latlng.lat,
                lon: latlng.lng
            });

            angle += spiralFootSeparation / r;
        }

        return res;
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
