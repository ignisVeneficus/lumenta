document.addEventListener('DOMContentLoaded', () => {
    const form = document.querySelector('form');

    form.addEventListener('submit', (e) => {
        packData(); 
    });
});

function packData(){
    document.querySelectorAll('.xrules').forEach(rules=>{
        rules.xrules.writeRules();
    });
}