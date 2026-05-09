function getParentId(ul) {
  const li = ul.closest(".album-node");
  return li ? parseInt(li.dataset.id) : null;
}
function getOrderedIds(ul) {
  return [...ul.children].map(li => parseInt(li.dataset.id));
}

function onDrop(evt) {
  const sourceUL = evt.from;
  const targetUL = evt.to;

  const sourceParent = getParentId(sourceUL);
  const targetParent = getParentId(targetUL);

  const sourceOrder = getOrderedIds(sourceUL);
  const targetOrder = getOrderedIds(targetUL);

  console.log({
    sourceParent,
    sourceOrder,
    targetParent,
    targetOrder
  });
}