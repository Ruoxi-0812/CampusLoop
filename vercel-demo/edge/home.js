(function () {
  'use strict';
  const grid = document.getElementById('campusloop-listings-grid');
  const sampleOrder = new Map(Array.from(grid.children, (el, index) => {
    const id = decodeURIComponent(el.querySelector('a').getAttribute('href').split('/').pop());
    el.dataset.listingId = id;
    return [id, index];
  }));
  const search = document.getElementById('campusloop-search-input');
  const message = document.getElementById('connection-message');
  const retry = document.getElementById('connection-retry');
  const params = new URLSearchParams(location.search);
  let category = params.get('category') || 'all';
  search.value = params.get('q') || '';
  function filters() {
    let count = 0;
    for (const card of grid.children) {
      card.hidden = !(card.textContent.toLowerCase().includes(search.value.trim().toLowerCase()) && (category === 'all' || card.dataset.categories.split(/\s+/).includes(category)));
      if (!card.hidden) count++;
    }
    document.querySelector('.campusloop-no-results').hidden = count !== 0;
    document.querySelectorAll('[data-category-filter]').forEach(control => {
      const active = control.dataset.categoryFilter === category;
      control.classList.toggle('active', active);
      if (control.tagName === 'BUTTON') control.setAttribute('aria-pressed', String(active));
    });
  }
  search.addEventListener('input', filters);
  document.querySelectorAll('[data-category-filter]').forEach(control => control.addEventListener('click', () => { category = control.dataset.categoryFilter; filters(); }));
  function node(tag, text, cls) { const el = document.createElement(tag); el.textContent = text; if (cls) el.className = cls; return el; }
  function card(listing) {
    const metadata = listing.metadata || {};
    const outer = node('div', '', 'col-md-4 hot-product-card'); outer.dataset.categories = metadata.category || 'other'; outer.dataset.listingId = listing.id;
    const inner = node('div', '', 'campusloop-card');
    const link = node('a', '', 'campusloop-card-media'); link.href = '/product/' + encodeURIComponent(listing.id);
    const img = document.createElement('img'); img.alt = listing.title; img.loading = 'lazy';
    // Only same-origin uploaded images and bundled assets are supported.
    img.src = /^\/(static\/|api\/marketplace\/listings\/)/.test(metadata.image || '') ? metadata.image : '/static/icons/listing-no-photo.svg';
    link.append(img, node('span', '$' + (listing.price_cents / 100).toFixed(2), 'campusloop-price-pill'), node('span', listing.seller_id === 'campusloop-demo-seller' ? 'Sample item' : listing.status, 'campusloop-status-pill'));
    const body = node('div', '', 'campusloop-card-body'); const footer = node('div', '', 'campusloop-card-footer');
    footer.append(node('span', listing.seller_id === 'campusloop-demo-seller' ? 'Northeastern' : (metadata.campus || 'Northeastern')), node('span', 'View item'));
    body.append(node('div', listing.title, 'hot-product-card-name'), footer); inner.append(link, body); outer.append(inner); return outer;
  }
  async function load() {
    message.parentElement.hidden = true;
    retry.hidden = true; message.textContent = 'Connecting to the marketplace… You can browse sample items while we connect.';
    try {
      const result = await CampusLoopConnection('/api/marketplace/listings', (text, response) => {
        if (!(response.headers.get('content-type') || '').includes('application/json')) return false;
        try { return Array.isArray(JSON.parse(text)); } catch (_) { return false; }
      });
      const listings = JSON.parse(result.text);
      try { sessionStorage.setItem('campusloop-listing-preview', JSON.stringify(listings)); } catch (_) {}
      // Keep bundled samples in their initial order; retain API order for new listings.
      listings.sort((a, b) => (sampleOrder.get(a.id) ?? -1) - (sampleOrder.get(b.id) ?? -1));
      const existing = new Map(Array.from(grid.children, el => [el.dataset.listingId, el]));
      listings.forEach((listing, index) => {
        let el = existing.get(listing.id);
        const fresh = card(listing);
        if (el) {
          existing.delete(listing.id);
          if (!sampleOrder.has(listing.id)) el.dataset.categories = fresh.dataset.categories;
          for (const selector of ['.hot-product-card-name', '.campusloop-price-pill', '.campusloop-status-pill', '.campusloop-card-footer span']) {
            const target = el.querySelector(selector), value = fresh.querySelector(selector).textContent;
            if (target.textContent !== value) target.textContent = value;
          }
          const img = el.querySelector('img'), nextImg = fresh.querySelector('img');
          if (img.getAttribute('src') !== nextImg.getAttribute('src')) img.src = nextImg.getAttribute('src');
          if (img.alt !== nextImg.alt) img.alt = nextImg.alt;
        } else el = fresh;
        if (grid.children[index] !== el) grid.insertBefore(el, grid.children[index] || null);
      });
      existing.forEach(el => el.remove());
      filters();
      message.textContent = '';
      message.parentElement.hidden = true;
    } catch (error) { message.parentElement.hidden = false; message.textContent = error.message + ' Sample items are still available to browse.'; retry.hidden = false; }
  }
  // Account state uses the same session as the real backend pages, never demo localStorage.
  try {
    const user = sessionStorage.getItem('campusloop-marketplace-token') && JSON.parse(sessionStorage.getItem('campusloop-marketplace-user'));
    if (user) {
      const signin = document.querySelector('[data-auth-signin]'); if (signin) { signin.textContent = 'My listings'; signin.href = '/my-listings'; }
      document.getElementById('campusloop-sign-in').textContent = 'My listings'; document.getElementById('campusloop-sign-in').href = '/my-listings';
    }
  } catch (_) {}
  document.querySelectorAll('a[href="/cart"]').forEach(link => { link.href = '/my-listings'; link.setAttribute('aria-label', 'My listings'); });
  retry.onclick = load; filters(); load();
}());
