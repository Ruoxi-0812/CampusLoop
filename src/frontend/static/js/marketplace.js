(function () {
  'use strict';
  const base = window.CampusLoopBackendConfig.baseUrl;
  const tokenKey = 'campusloop-marketplace-token';
  const userKey = 'campusloop-marketplace-user';
  function getUser() { try { return sessionStorage.getItem(tokenKey) ? JSON.parse(sessionStorage.getItem(userKey)) : null; } catch (_) { return null; } }
  async function api(path, options = {}) {
    const headers = { 'Content-Type': 'application/json' };
    const token = sessionStorage.getItem(tokenKey);
    if (token) headers.Authorization = 'Bearer ' + token;
    if (options.key) headers['Idempotency-Key'] = options.key;
    const response = await fetch(base + '/api/marketplace' + path, { method: options.method || 'GET', headers, body: options.body === undefined ? undefined : JSON.stringify(options.body) });
    const data = await response.json();
    if (!response.ok) { const error = new Error(data.error || 'Could not complete request'); error.status = response.status; throw error; }
    return data;
  }
  function show(message) {
    let status = document.getElementById('campusloop-workflow-status');
    if (!status) { status = document.createElement('p'); status.id = 'campusloop-workflow-status'; status.className = 'campusloop-workflow-status'; status.setAttribute('role', 'status'); status.setAttribute('aria-live', 'polite'); const main = document.querySelector('main'); main.prepend(status); }
    status.textContent = message;
    status.hidden = !message;
  }
  async function logout() { try { await api('/logout', {method: 'POST'}); } catch (e) { if (e.status !== 401) { show(e.message); return; } } sessionStorage.removeItem(tokenKey); sessionStorage.removeItem(userKey); location.assign(base + '/'); }
  window.CampusLoopBackend = { getUser, api, logout };
  function element(tag, text, className) { const node = document.createElement(tag); node.textContent = text; if (className) node.className = className; return node; }
  function action(label, fn) { const button = element('button', label, 'campusloop-button campusloop-button-secondary'); button.type = 'button'; button.addEventListener('click', async () => { button.disabled = true; show(''); try { await fn(); } catch (e) { show(e.message); } finally { button.disabled = false; } }); return button; }
  function requireUser() { if (getUser()) return true; location.assign(base + '/login?next=' + encodeURIComponent(location.pathname)); return false; }
  function safeNext(raw) { try { const url = new URL(raw || base + '/', location.origin); if (url.origin === location.origin && url.pathname.startsWith(base + '/')) return url.pathname + url.search; } catch (_) {} return base + '/'; }
  function reservationKey(id) { return 'campusloop-reserve:' + getUser().id + ':' + id; }
  async function loadDashboard() {
    if (!requireUser()) return;
    const [listings, reservations] = await Promise.all([api('/listings'), api('/reservations')]);
    const user = getUser();
    const target = document.querySelector('[data-my-listings-page-list]'); target.replaceChildren();
    const owned = listings.filter(l => l.seller_id === user.id);
    document.querySelector('[data-empty-listings]').hidden = owned.length !== 0;
    for (const l of owned) { const row = element('div', '', 'campusloop-my-listing'); const details = element('div', ''); const link = element('a', l.title); link.href = base + '/product/' + encodeURIComponent(l.id); details.append(link, element('span', l.status + ' · ' + l.pickup), element('small', [l.metadata.campus, l.metadata.handoff].filter(Boolean).join(' · '))); row.append(details, element('b', '$' + (l.price_cents / 100).toFixed(2))); target.append(row); }
    const panel = document.querySelector('[data-reservation-list]'); panel.replaceChildren();
    if (!reservations.length) panel.append(element('p', 'Your reservations and buyer requests will appear here.', 'campusloop-empty-listings'));
    for (const r of reservations) {
      const l = listings.find(l => l.id === r.listing_id);
      const row = element('div', '', 'campusloop-my-listing'); const details = element('div', ''); const link = element('a', l ? l.title : 'View item'); link.href = base + '/product/' + encodeURIComponent(r.listing_id);
      details.append(link, element('span', (r.buyer_id === user.id ? 'Your reservation' : 'Buyer request') + ' · ' + r.status)); row.append(details);
      if (r.status === 'active') { const op = r.buyer_id === user.id ? 'cancel' : 'complete'; row.append(action(op === 'cancel' ? 'Cancel reservation' : 'Confirm handoff', async () => { await api('/reservations/' + r.id + '/' + op, {method:'POST'}); sessionStorage.removeItem(reservationKey(r.listing_id)); await loadDashboard(); show(op === 'cancel' ? 'Reservation cancelled. The item is available again.' : 'Handoff confirmed. Item marked sold.'); })); }
      panel.append(row);
    }
  }
  async function setupProduct() {
    const button = document.querySelector('[data-reserve-listing]'); if (!button) return;
    const id = button.dataset.reserveListing;
    const listing = await api('/listings/' + encodeURIComponent(id));
    const status = document.querySelector('[data-listing-status]'); status.textContent = listing.status;
    const user = getUser();
    let active;
    if (user) { const reservations = await api('/reservations'); active = reservations.find(r => r.listing_id === id && r.status === 'active'); const old = reservations.find(r => r.listing_id === id && r.buyer_id === user.id); if (old && old.status !== 'active') sessionStorage.removeItem(reservationKey(id)); }
    button.disabled = false;
    if (user && listing.seller_id === user.id) { button.textContent = 'Manage listing'; button.onclick = () => location.assign(base + '/my-listings'); return; }
    if (active) { button.textContent = 'View reservation'; button.onclick = () => location.assign(base + '/my-listings'); return; }
    if (listing.status !== 'available') { button.textContent = listing.status === 'sold' ? 'Sold' : 'Already reserved'; button.disabled = true; return; }
    button.textContent = 'Reserve item';
    button.onclick = async () => {
      if (!requireUser()) return;
      button.disabled = true; show('');
      const storageKey = reservationKey(id); let key = sessionStorage.getItem(storageKey); if (!key) { key = crypto.randomUUID(); sessionStorage.setItem(storageKey, key); }
      try { await api('/listings/' + id + '/reserve', {method:'POST', key}); await setupProduct(); show('Item reserved. View your reservation in My listings.'); }
      catch (e) { show(e.message); if (e.status === 409) await setupProduct(); else button.disabled = false; }
    };
  }
  document.addEventListener('DOMContentLoaded', async () => {
    try {
      if (sessionStorage.getItem(tokenKey)) { try { const me = await api('/me'); sessionStorage.setItem(userKey, JSON.stringify({id:me.user_id,email:me.email,name:me.name||me.email,role:'seller'})); } catch (e) { if (e.status !== 401) throw e; sessionStorage.removeItem(tokenKey); sessionStorage.removeItem(userKey); } }
      if (window.CampusLoopAuth) window.CampusLoopAuth.update();
      const auth = document.getElementById('campusloop-signin-page-form');
      if (auth) {
        const buttons = Array.from(document.querySelectorAll('[data-auth-mode]')); const submit = auth.querySelector('[data-auth-submit]');
        let mode = location.pathname.endsWith('/signup') ? 'signup' : 'login';
        function switchMode(value) { mode = value; buttons.forEach(b => { b.classList.toggle('active', b.dataset.authMode === mode); b.setAttribute('aria-pressed', String(b.dataset.authMode === mode)); }); submit.textContent = mode === 'signup' ? 'Create account' : 'Log in'; const name = auth.elements.name; name.required = mode === 'signup'; name.closest('label').hidden = mode !== 'signup'; auth.elements.password.autocomplete = mode === 'signup' ? 'new-password' : 'current-password'; }
        buttons.forEach(b => b.onclick = () => switchMode(b.dataset.authMode)); switchMode(mode);
        auth.onsubmit = async e => { e.preventDefault(); submit.disabled = true; show(''); try { const form = new FormData(auth); const result = await api('/auth/' + (mode === 'signup' ? 'register' : 'login'), {method:'POST',body:{email:String(form.get('email')),password:String(form.get('password')),name:String(form.get('name')||'')}}); sessionStorage.setItem(tokenKey,result.token); sessionStorage.setItem(userKey,JSON.stringify({id:result.user_id,email:result.email,name:result.name||result.email,role:'seller'})); location.assign(safeNext(new URLSearchParams(location.search).get('next'))); } catch (e) { show(e.message); } finally { submit.disabled = false; } };
      }
      const post = document.getElementById('campusloop-post-form');
      if (post && requireUser()) post.onsubmit = async e => {
        e.preventDefault(); const button = post.querySelector('button[type="submit"]'); button.disabled = true; show('');
        try {
          const values = new FormData(post); const metadata = {}; for (const key of ['category','city','campus','dorm','address','contact','handoff']) metadata[key] = String(values.get(key)||'').trim();
          const file = post.elements.image.files[0]; if (file) { if (file.size > 2*1024*1024 || !['image/png','image/jpeg','image/webp','image/gif'].includes(file.type)) throw new Error('Choose a PNG, JPEG, GIF or WebP image smaller than 2 MB.'); metadata.image = await new Promise((resolve,reject) => { const reader = new FileReader(); reader.onload = () => resolve(reader.result); reader.onerror = () => reject(new Error('Could not read image')); reader.readAsDataURL(file); }); }
          const l = await api('/listings',{method:'POST',body:{title:String(values.get('name')).trim(),description:String(values.get('description')).trim(),price_cents:Math.round(Number(values.get('price'))*100),pickup:String(values.get('pickup')).trim(),metadata}});
          location.assign(base + '/product/' + l.id);
        } catch (e) { show(e.message); button.disabled = false; }
      };
      if (document.querySelector('[data-reservation-list]')) await loadDashboard();
      await setupProduct();
    } catch (e) { show(e.message); }
  });
}());
