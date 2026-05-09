(function () {
    // ============================================
    // DOM INIT
    // ============================================

    document.addEventListener('DOMContentLoaded', () => {
        document.querySelectorAll('.xselect').forEach(initFromDOM);
    });

    function initFromDOM(root, config=null) {
        if(!config){
            config = {
                mode: root.dataset.mode,
                name: root.dataset.name,
                src: root.dataset.src,
                i18nId: root.dataset.i18nMap,
                mapping: {
                    idKey: root.dataset.idKey,
                    labelKey: root.dataset.labelKey,
                    childrenKey: root.dataset.childrenKey,
                    pillLabelKey: root.dataset.pillLabelKey,
                    i18nKey: root.dataset.i18nKey,
                },
                search: root.dataset.search !== "false",
                t: window.t
            };
            let i18n = {};
            if (config.i18nId) {
                const node = document.getElementById(id);
                if (node) {
                    i18n = JSON.parse(node.textContent);
                }
            }
            config.i18n = i18n;

            // inline selected data JSON support
            const selectedNode = root.querySelector('.xselect-selected');
            if (selectedNode) {
                config.selected = parseJSON(selectedNode.textContent);
            }

            // inline data JSON support
            const dataNode = root.querySelector('.xselect-data');
            if (dataNode) {
                console.debug(dataNode);
                config.data = normalizeData(parseJSON(dataNode.textContent), config);
                console.debug(config.data);
            }
        }
        const ctrl = coreInit(root, config);

        if (!config.data && config.src) {
            loadAndSetData(ctrl, config.src);
        }
    }

    // ============================================
    // JS INIT (PUBLIC API)
    // ============================================

    async function initXSelect(root, config) {
        const ctrl = coreInit(root, config);
        if (!config.data && config.src) {
            await loadAndSetData(ctrl, config.src);
        }
        return ctrl;
    }

    function destroyXSelect(root) {
        root._xselect?.destroy?.();
    }

    // export
    window.XSelect = {
        init: initXSelect,
        destroy: destroyXSelect,
    };

    // ============================================
    // CORE INIT (PURE)
    // ============================================

    function coreInit(root, config) {
        // destroy previous instance
        if (root.xselect?.destroy) {
            root.xselect.destroy();
        }

        const state = {
            root,
            mapping: config.mapping,
            mode: config.mode || 'single',
            name: config.name || '',
            data: (config.data || []),
            displayData: (config.data || []),
            selectedNodes: new Set(),
            selected: (config.selected || []),
            query: '',
            isLoading: !config.data,
            nodeIds: new Map(),
            search: config.search,
            i18n: config.i18n,
            t: config.t
        };

        createNodeList(state);

        const controller = createController(state);
        root.xselect = controller;

        // DOM build
        buildLayout(root, state);
        // first render
        render(controller, state);

        return controller;
    }

    // ============================================
    // CONTROLLER
    // ============================================

    function createController(state) {
        return {
            setData(data) {
                state.data = normalizeData(data, state.mapping, state.i18n, state.t);
                state.displayData = state.data;
                state.isLoading = false;
                createNodeList(state);

                render(this, state);
            },

            selected(node) {
                selectNode(this, state, node);
                if (node._input) {
                    node._input.checked = true;
                }
                renderPills(this, state);
                renderInput(this, state);
                if (state.mode == "single") {
                    this.close();
                }
                state.root.dispatchEvent(new CustomEvent("xselect:select", {
                detail: { node }
                }));
          
            },
            deselect(node) {
                state.selectedNodes.delete(node);
                if (node._input) {
                    node._input.checked = false;
                }
                renderPills(this, state);
                renderInput(this, state);
                state.root.dispatchEvent(new CustomEvent("xselect:deselect", {
                detail: { node }
                }));

            },
            open() {
                closeAll();
                updateOpenState(state,true);
            },

            close() {
                updateOpenState(state,false);
            },
            toggle(){
                const isOpen = state.nodes.dropdown.classList.contains("open");
                if(!isOpen){
                    this.open();
                } else {
                    this.close();
                }
            },
            destroy() {
                cleanup(state);
            },

            getState() {
                return state;
            },

            search(query) {
                state.query = query || '';
                if (!query) {
                    state.displayData = state.data;
                } else {
                    state.displayData = filterTree(state.data, query);
                }
                renderTree(this, state);
            },
            getSelected(){
                let list = [...state.selectedNodes].map(n => n.id);
                if (state.mode == 'single') return (list[0]||null);
                return list;
            },
            getSelectedNodes(){
                return [...state.selectedNodes];
            },
            setSelected(selected){
                selected =Array.isArray(selected)?selected:[selected];
                state.selected = selected;
                createSelected(state);
            }
        };
    }


    // ============================================
    // DATA LOAD
    // ============================================

    async function loadAndSetData(ctrl, url) {
        try {
            const data = await loadData(url);
            if (data.data) {
                ctrl.setData(data.data);
            }
        } catch (e) {
            console.error('xselect: load failed', e);
        }
    }


    async function loadData(url) {
        const res = await fetch(url);
        if (!res.ok) {
            throw new Error('failed to load ' + url);
        }
        return await res.json();
    }


    // ============================================
    // RENDER
    // ============================================

    function render(ctrl) {
        const state = ctrl.getState();
        renderTree(ctrl, state);
        renderPills(ctrl, state);
    }
    function renderNode(ctrl, node, selectedNodes, masterName, state) {
        const ret = document.createElement('li');
        ret.className = "xselect tree-node";
        const row = document.createElement('div');
        row.className = "xselect-tree-row";
        if (node.children?.length) {
            var toggle = document.createElement('input');
            toggle.type = "checkbox";
            toggle.id = masterName + "TG" + node.id;
            toggle.className = "toggle tree-toggle";
            toggle.checked = true;
            var toggleLabel = document.createElement('label');
            toggleLabel.htmlFor = toggle.id;
            toggleLabel.className = "xselect-toggle-label tree-label";
            toggleLabel.innerHTML = `<i class="fa-solid fa-chevron-right xselect-icon tree-icon icon"></i>`;
            row.appendChild(toggle);
            row.appendChild(toggleLabel);
        } else {
            const place = document.createElement('div');
            place.className = "xselect-placeholder";
            row.appendChild(place);
        }
        var selection = document.createElement('input');
        selection.type = "checkbox";
        //selection.name = masterName + "CB" + node.id;
        selection.id = masterName + "CB" + node.id;
        selection.className = "selection";
        if (selectedNodes.has(node)) {
            selection.checked = true;
        }
        selection.addEventListener('change', (e) => {
            if (e.target.checked) {
                ctrl.selected(node);
            } else {
                ctrl.deselect(node);
            }
        });
        node._input = selection;
        var selectionLabel = document.createElement('label');
        selectionLabel.htmlFor = selection.id;
        selectionLabel.className = "tree-label xselect-selection-label";
        selectionLabel.textContent = node.display;
        row.appendChild(selection);
        row.appendChild(selectionLabel);
        ret.appendChild(row);
        const frag = document.createDocumentFragment();
        if (node.children?.length) {
            frag.appendChild(renderNodes(ctrl, node.children, selectedNodes, masterName, state));
        }
        ret.appendChild(frag);
        return ret;
    }

    function renderNodes(ctrl, nodes, selectedNodes, masterName) {
        var ret = document.createElement('ul');
        ret.className = `xselect-tree tree`;
        nodes.forEach(node => {
            ret.appendChild(renderNode(ctrl, node, selectedNodes, masterName));
        })

        return ret;
    }
    function renderTree(ctrl, state) {
        const nodes = state.nodes;
        const tree = nodes.tree;
        tree.innerHTML = '';
        const masterName = state.name==""?crypto.randomUUID():state.name;
        tree.appendChild(renderNodes(ctrl, state.displayData, state.selectedNodes, masterName))
    }
    function renderPills(ctrl, state) {
        const selected = [...state.selectedNodes].sort((a, b) => a.display.localeCompare(b.display));
        const nodes = state.nodes;
        /* if (state.mode == "single") {
            const node = nodes.single
            node.innerHTML = (selected[0] ? selected[0].pillLabelKey : "");
        }
        else { */
            const parent = nodes.pills
            parent.innerHTML = '';
            selected.forEach(node => {
                var line = document.createElement('div');
                line.className = `xselect-pill${node.error?" error":""}`;
                var label = document.createElement('div');
                label.className = `xselect-pill-label`;
                label.textContent = node.pill;
                var deleteBtn = document.createElement('button');
                deleteBtn.className = "xselect-delete action";
                deleteBtn.innerHTML = `<i class="fa-solid fa-xmark icon"></i>`
                deleteBtn.type = "button";
                deleteBtn.addEventListener('click', (e) => {
                    e.stopPropagation();
                    ctrl.deselect(node);
                });
                line.appendChild(label);
                line.appendChild(deleteBtn);
                parent.appendChild(line);
            })
        //}
    }

    function renderInput(ctrl, state) {
        const selected = [...state.selectedNodes];
        const parent = state.nodes.hidden;
        parent.innerHTML = '';
        selected.forEach(node => {
            var line = document.createElement('input');
            line.className = 'xselect-output';
            line.type = 'hidden';
            line.value = node.id;
            line.name = state.name;
            parent.appendChild(line);
        })
    }

    // ============================================
    // LAYOUT
    // ============================================

    function buildLayout(root, state) {
        root.innerHTML = '';

        const control = document.createElement('div');
        control.className = `xselect-control ${state.mode}`;

        let pills = null;
        let single = null;

        // if (state.mode === 'multiple') {
            pills = document.createElement('div');
            pills.className = 'xselect-pills';
            control.appendChild(pills);
        /*
        } else {
            single = document.createElement('div');
            single.className = 'xselect-single-value';
            single.textContent = 'Select...';
            control.appendChild(single);
        }
        */

        const iconWrap = document.createElement('button');
        iconWrap.type = "button";
        iconWrap.className = 'xselect-icon action';

        const icon = document.createElement('i');
        icon.className = 'fa-solid fa-caret-down icon';
        iconWrap.appendChild(icon);
        control.appendChild(iconWrap);

        const dropdown = document.createElement('div');
        dropdown.className = 'xselect-dropdown overlay';

        var search = null;
        if (state.search){
            search = document.createElement('input');
            search.className = 'xselect-search';
            search.type = 'text';
            search.placeholder = 'Search...';
            //TODO: add trigger
            dropdown.appendChild(search);
        }

        const tree = document.createElement('div');
        tree.className = 'xselect-tree-wrapper';


        dropdown.appendChild(tree);

        const hidden = document.createElement('div');
        hidden.className = 'xselect-hidden';

        root.appendChild(control);
        root.appendChild(dropdown);
        root.appendChild(hidden);

        state.nodes = {
            pills: pills,
            single: single,
            dropdown: dropdown,
            search: search,
            tree: tree,
            hidden: hidden
        };
        control.addEventListener("click", (e) => {
            if (e.target.closest(".xselect-pill")) {
                return;
            }
            e.stopPropagation();
            root.xselect.toggle();
        });

    }


    // ============================================
    // HELPERS
    // ============================================

    function translate(val, prefix, postfix, t) {
        if (!t || !prefix) return val;

        const key = prefix + "." + val +(postfix)?("." +postfix):"";
        const tr = t(key);

        return tr && !tr.startsWith("[") ? tr : val;
    }

    function mapNode(n, mapping, i18n, t) {
        const labelKey = mapping.labelKey || 'display';
        const pillKey  = mapping.pillLabelKey;

        let display = String(n[labelKey]);
        let pill    = pillKey ? n[pillKey] : display;

        // 🔹 i18n mapping
        if (i18n && t) {
            display = translate(display, mapping.i18nKey, "display", t);
            pill    = translate(pill,    mapping.i18nKey, "short", t);
        }

        return {
            id: String(n[mapping.idKey || 'id']),
            display: display,
            pill: String(pill).replaceAll('/', '/\u200B'),
            children: (n[mapping.childrenKey || 'children'] || [])
                .map(child => mapNode(child, mapping, i18n, t)),
        };
    }

    function normalizeData(data, mapping, i18n, t) {
        if (!Array.isArray(data)) return [];
        return (data || []).map(n => mapNode(n, mapping, i18n, t));
    }

    function parseJSON(str) {
        try {
            return JSON.parse(str);
        } catch {
            return null;
        }
    }

    function selectNode(controller, state, node) {
        if (state.mode == "single") {
            state.selectedNodes.forEach(node => controller.deselect(node));
            state.selectedNodes.clear();
        }
        state.selectedNodes.add(node);
    }

    function closeAll(){
        document.querySelectorAll(".xselect-dropdown.open")
            .forEach(el => el.classList.remove("open"));
    }

    function updateOpenState(state,open) {
       var dropdown = state.nodes.dropdown
        if (open) {
            dropdown.classList.add("open");
        } else {
            dropdown.classList.remove("open");
        }
    }

    function indexNodes(nodes, map) {
        nodes.forEach(n => {
            map.set(n.id, n);
            if (n.children) indexNodes(n.children, map);
        });
    }

    function createSelected(state){
        state.selectedNodes = new Set();
        (state.selected || []).forEach(id => {
            const node = state.nodeIds.get(String(id));
            if (node) {
                state.selectedNodes.add(node);
            } else {
                state.selectedNodes.add({
                    id:id,
                    pill:id,
                    error:true,
                    display:id,
                })
            }
        });
    }

    function createNodeList(state){
        state.nodeIds = new Map();
        if (state.data.length > 0) {
            indexNodes(state.data, state.nodeIds);
            createSelected(state);
        }
    }

    function cleanup(state) {
        // TODO:
        // - remove event listeners
        // - clear DOM if needed
        state.root.innerHTML = '';
    }


})();