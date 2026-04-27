(function () {
    document.addEventListener('DOMContentLoaded', () => {
        document.querySelectorAll('.xrules').forEach(initRules);
    });

    function initRules(root, conf = null) {
        if(!conf){
            const outputInputId= root.dataset.outputInput;
            const rulesElemId = root.dataset.rulesElem;
            conf = {
                outputInput: document.getElementById(outputInputId),
                dataNode: document.getElementById(rulesElemId),
                tagsUrl: root.dataset.tagsUrl,
                albumsUrl:root.dataset.albumsUrl,
                root:root,
            };
        }
        loadInitialState(conf);

        coreInit(conf);
    }

    function loadInitialState(conf) {
        if (!conf){
            return;
        }
        const node = conf.dataNode;
        if (!node){
            conf.rules=createEmptyGroup();
            return;
        }
        try {
            if(node && node.textContent){
                const parsed = JSON.parse(node.textContent);
                conf.rules= normalizeGroup(parsed);
            }
            else {
                conf.rules= createEmptyGroup();
            }
        } catch (e) {
            console.warn('Invalid rules JSON', e);
            conf.rules= createEmptyGroup();
        }
    }

    function createEmptyGroup() {
        return {
            op: 'all',
            rules: []
        };
    }

    function normalizeGroup(group) {
        return {
            op: group?.op || 'all',
            rules: Array.isArray(group?.rules) ? group.rules : []
        };
    }    

    function coreInit(conf){
        conf.abstractRules = getAbstractRules();
        conf.ruleIds = new Map();
        conf.abstractRules.forEach(r => {
            conf.ruleIds.set(r.id, r);
        });
        buildLayout(conf);
        fillLayout(conf);
        controller = createController(conf);
        conf.root.xrules = controller;
    }

    //-----------------------
    function createController(conf) {

        function writeRules() {
            const ret = {op:"",rules:[]}
            const groupOp = conf.root.querySelector('.rule-group-op');
            if(groupOp){
                let value=groupOp.xselect.getSelected();
                ret.op = value;
            }
            conf.root.querySelectorAll('.xrule-data').forEach(row => {
                let data={};
                row.querySelectorAll('[data-name]').forEach(piece=>{
                    let name = piece.dataset.name;
                    let value = getValue(piece);
                    data[name]=value;
                });

                ret.rules.push(data);
            });
            let text=JSON.stringify(ret);
            conf.outputInput.value = text;
        }

        return {
            writeRules
        };
    }

    function getValue(node){
        let type= node.dataset.type;
        switch(type){
            case "const":
                return node.dataset.value;
            case "xselect":
                return node.xselect.getSelected();
            case "string":
                return node.value;
            case "checkbox":
                return node.checked;
        }
        return null;
    }

    function setValue(node,value){
        let type= node.dataset.type;
        switch(type){
            case "xselect":
                node.xselect.setSelected(value);
            case "string":
                node.value=value;
            case "checkbox":
                node.checked=!!value
        }

    }

    function buildLayout(conf){
        const root = conf.root;
        root.innerHTML = '';

        let row=document.createElement('div');
        row.className = `xrule-row`;

        let label=document.createElement('div');
        //label.htmlFor= "groupOp-"+rootId;
        label.textContent = 'Match rules:';
        label.className = `xrule-label`;

        const groupOp = document.createElement('div');
        groupOp.className = `xselect rule-group-op`;
        groupOp.dataset.type = `xselect`;
        groupOp.dataset.name = `op`;
        //groupOp.id = "groupOp-"+rootId;
        XSelect.init(groupOp, {
            mode:"single",
            name:"",
            data:[
                {   id:"all",
                    display:"All rules",
                    pill:"All rules"
                },
                {   id:"any",
                    display:"Any rules",
                    pill:"Any rules"
                }],
            selected:["all"]
        });
        row.appendChild(label);
        row.appendChild(groupOp);

        root.appendChild(row);

        row=document.createElement('div');
        row.className = `xrule-row`;
        
        label=document.createElement('div');
        label.textContent = 'New rule:';
        label.className = `xrule-label`;

        const typeSelect = document.createElement('div');
        typeSelect.className = `xselect rule-type-grp`;
        typeSelect.dataset = `xselect`;
        XSelect.init(typeSelect, {
            mode:"single",
            name:"",
            data:conf.abstractRules,
            selected:[],
        });

        const addButton = document.createElement('button');
        addButton.className = `xrule rule-add action`;
        addButton.type = "button";
        addButton.innerHTML = `<i class="fa-solid fa-plus xrule-icon icon" title="Add new rule"></i>`;
        addButton.addEventListener('click', (e) => {
                    e.stopPropagation();
                    addNewRule(conf,typeSelect.xselect.getSelectedNodes())
                });

        const wrapper = document.createElement('div');
        wrapper.className="xrule-wrapper";
        wrapper.appendChild(addButton);

        row.appendChild(label);
        row.appendChild(typeSelect);
        row.appendChild(wrapper);

        root.appendChild(row);


        conf.nodes= {
            groupOp:groupOp,
            newSelect:typeSelect,
            lastRow:row,
        }
    }
    function addNewRule(conf,ruleTypes){
        const root=conf.root;
        const lastRow = conf.nodes.lastRow;
        const ruleType =Array.isArray(ruleTypes) ? ruleTypes[0] : ruleTypes;
        if (!ruleType) return null;
        const rule = (ruleType.id || null);
        if (rule == null) return null;
        const labelTxt = ruleType.pill;

        const row= document.createElement('div');
        row.className = `xrule-row xrule-data`;
 
        const label=document.createElement('div');
        label.textContent = labelTxt;
        label.className = `xrule-label`;
        label.dataset.type = `const`;
        label.dataset.name = `type`;
        label.dataset.value = rule;

        row.appendChild(label);
        const selectData=getOp(rule);
        if(selectData!=null){
            const optSelect= document.createElement('div');
            optSelect.className = `xselect rule-type-grp`;
            optSelect.dataset.type = `xselect`;
            optSelect.dataset.name = `op`;
            XSelect.init(optSelect, {
                mode:"single",
                name:"",
                data: selectData
            });
            row.appendChild(optSelect);
        } else{
            row.appendChild(document.createElement('div'));
        }
        createRuleDataBlock(conf,row,rule);
        const deleteButton = document.createElement('button');
        deleteButton.className = `xrule rule-delete action`;
        deleteButton.type = "button";
        deleteButton.innerHTML = `<i class="fa-solid fa-xmark xrule-icon icon" title="Delete rule"></i>`;
        deleteButton.addEventListener('click', (e) => {
                    e.stopPropagation();
                    row.remove();
                });

        const wrapper = document.createElement('div');
        wrapper.className="xrule-wrapper";
        wrapper.appendChild(deleteButton);

        row.appendChild(wrapper);

        root.insertBefore(row,lastRow);
        return row;
    }

    function fillLayout(conf){
        const rules = conf.rules;
        const groupOp = conf.root.querySelector('.rule-group-op');
        if(groupOp){
            groupOp.xselect.setSelected(rules.op || "all");
        }
        rules.rules.forEach(rule=>{
            if(rule.type){
                let ruleData = conf.ruleIds.get(rule.type);
                if (!ruleData){
                    return;
                }
                row = addNewRule(conf,ruleData);
                row.querySelectorAll('[data-name]').forEach(piece=>{
                    let name = piece.dataset.name;
                    if(rule[name]){
                        setValue(piece,rule[name]);
                    }
                });
            }
        })
    }

    function getOp(type){
        switch(type){
            case "tag":
            case "path":
            case "extension":
            case "album":
                return [
                    {
                        id: "all",
                        display: "All of them",
                        pill: "All"
                    },
                    {
                        id: "any",
                        display: "One of them",
                        pill: "One"
                    },
                    {
                        id: "none",
                        display: "None of them",
                        pill: "None"
                    },
                    {
                        id: "only",
                        display: "Only them",
                        pill: "only"
                    },
                ];
            case "date":
                return [
                    {
                        id: "on",
                        display: "On date",
                        pill: "On"
                    },
                    {
                        id: "before",
                        display: "Before",
                        pill: "Before"
                    },
                    {
                        id: "after",
                        display: "After",
                        pill: "After"
                    },
                ];
            case "rating":
            case "width":
            case "height":
            case "aspect":
                return [
                    {
                        id: "<",
                        display: "<",
                        pill: "<"
                    },
                    {
                        id: ">",
                        display: ">",
                        pill: ">"
                    }
                ];
            case "name":
            case "notchildren":
                return null;

        }
    }
    function createRuleDataBlock(conf,row,rule){
        const block = document.createElement('div');
        block.className = `xrule-block xrule-block-${rule}`;
        switch(rule){
            case "tag":
                const tags = document.createElement('div');
                tags.className = `xselect rule-tags`;
                tags.dataset.type = `xselect`;
                tags.dataset.name = `tags`;
                XSelect.init(tags, {
                    mode:"multiple",
                    name:"",
                    src:conf.tagsUrl,
                    mapping: {
                        //idKey: "fullName",
                        idKey: "id",
                        labelKey: "name",
                        childrenKey: "children" ,
                        pillLabelKey: "fullName",
                    }
                });
                block.appendChild(tags);
                break;
            case "date":
                const dateInput = document.createElement('input');
                dateInput.className = `rule-date`;
                dateInput.dataset.type = `string`;
                dateInput.dataset.name = `date`;
                dateInput.placeholder="yyyy[.mm[.dd]]";
                block.appendChild(dateInput);
                break;
            case "name":
                const nameInput = document.createElement('input');
                nameInput.className = `rule-name`;
                nameInput.dataset.type = `string`;
                nameInput.dataset.name = `pattern`;
                nameInput.placeholder="regexp pattern";
                block.appendChild(nameInput);
                break;
            case "rating":
            case "width":
            case "height":
            case "aspect":
                const valueInput = document.createElement('input');
                valueInput.className = `rule-value`;
                valueInput.dataset.type = `string`;
                valueInput.dataset.name = `value`;
                switch(rule){
                    case "rating":valueInput.placeholder="3";break
                    case "aspect":valueInput.placeholder="1.2";break;
                    default:valueInput.placeholder="1234";
                }
                valueInput.type = "text";
                valueInput.inputMode = "decimal"
                block.appendChild(valueInput);
                break;
            case "path":
                let label= document.createElement('div');
                label.className = `xrule-label xrule-paths`;
                label.textContent="Root:";
                block.appendChild(label);

                const rootInput = document.createElement('input');
                rootInput.className = `rule-path`;
                rootInput.dataset.type = `string`;
                rootInput.dataset.name = `root`;
                rootInput.placeholder="holiday";
                block.appendChild(rootInput);

                label= document.createElement('div');
                label.className = `xrule-label xrule-paths`;
                label.textContent="Paths:";
                block.appendChild(label);

                const pathInput = document.createElement('input');
                pathInput.className = `rule-path`;
                pathInput.dataset.type = `string`;
                pathInput.dataset.name = `paths`;
                pathInput.placeholder="root/path/";
                block.appendChild(pathInput);
                break;
            case "extension":
                const extInput = document.createElement('input');
                extInput.className = `rule-extension`;
                extInput.dataset.type = `string`;
                extInput.dataset.name = `extensions`;
                extInput.placeholder=".jpg .tiff";
                block.appendChild(extInput);
                break;
            case "album":
                const albums = document.createElement('div');
                albums.className = `xselect rule-albums`;
                albums.dataset.type = `xselect`;
                albums.dataset.name = `albums`;
                XSelect.init(albums, {
                    mode:"multiple",
                    name:"",
                    src:conf.albumsUrl,
                    mapping: {
                        idKey: "fullName",
                        labelKey: "name",
                        childrenKey: "children" ,
                        pillLabelKey: "fullName",
                    }
                });
                block.appendChild(albums);                
                let chk = createCheckbox("include_children","Include children albums");
                block.appendChild(chk);

                break;
            case "notchildren":
                break;
  

        }
        row.appendChild(block);
    }

    function createCheckbox(name,label){
        const id= crypto.randomUUID();
        const wrap = document.createElement('div');
        wrap.className = 'xrule-toggle';

        const input = document.createElement('input');
        input.type = 'checkbox';
        input.id = id;
        input.hidden = true;
        input.dataset.type = `checkbox`;
        input.dataset.name = name;

        const labelNode = document.createElement('label');
        labelNode.className = 'xrule-toggle-label';
        labelNode.setAttribute('for', id);
        labelNode.innerHTML=`<div class="action"><i class="fa-regular fa-square icon icon-off" aria-hidden="true"></i>
        <i class="fa-regular fa-square-check icon icon-on" aria-hidden="true"></i></div>
        <span>${label}</span>`

        wrap.appendChild(input);
        wrap.appendChild(labelNode);

        return wrap;
    }
    function getAbstractRules(){
    return [
                {
                    id: "album",
                    display: "From selected albums",
                    pill: "Album"
                },
                {
                    id: "notchildren",
                    display: "Exclude in child albums",
                    pill: "Not in children"
                },
                {
                    id: "tag",
                    display: "With matching tags",
                    pill: "Tags"
                },
                {
                    id: "date",
                    display: "By date",
                    pill: "Date"
                },
                {
                    id: "name",
                    display: "By name pattern",
                    pill: "Name"
                },
                {
                    id: "rating",
                    display: "By rating",
                    pill: "Rating"
                },
                {
                    id: "path",
                    display: "By file path",
                    pill: "Path"
                },
                {
                    id: "extension",
                    display: "By file extension",
                    pill: "Ext"
                },
                {
                    id: "width",
                    display: "By width",
                    pill: "Width"
                },
                {
                    id: "height",
                    display: "By height",
                    pill: "Height"
                },
                {
                    id: "aspect",
                    display: "By aspect ratio",
                    pill: "Aspect"
                }
            ]
    }
})();