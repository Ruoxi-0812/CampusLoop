(function () {
  'use strict';
  const id = decodeURIComponent(location.pathname.split('/').filter(Boolean).pop());
  document.querySelector('[data-reserve-listing]').dataset.reserveListing = id;
  window.CampusLoopRenderProduct = function(item) {
    document.getElementById('preview-title').textContent = item.title;
    document.getElementById('preview-description').textContent = item.description;
    document.getElementById('preview-price').textContent = '$' + (item.price_cents / 100).toFixed(2);
    const image = document.getElementById('preview-image');
    image.src = /^\/(static\/|api\/marketplace\/listings\/)/.test(item.metadata?.image || '') ? item.metadata.image : '/static/icons/listing-no-photo.svg';
    image.alt = item.title; image.hidden = false;
    document.getElementById('preview-label').textContent = item.seller_id === 'campusloop-demo-seller' ? 'Sample item. Create an account to publish your own listing.' : '';
    document.getElementById('product-pickup').textContent = item.pickup || '';
    document.getElementById('product-contact').textContent = item.metadata?.contact || '';
    document.getElementById('product-handoff').textContent = [item.metadata?.campus,item.metadata?.handoff,item.metadata?.address].filter(Boolean).join(' · ');
  };
}());
