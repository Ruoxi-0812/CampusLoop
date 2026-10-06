(function () {
  'use strict';
  const backend = window.CampusLoopBackend;
  function el(tag, text, cls) { const n = document.createElement(tag); n.textContent = text; if (cls) n.className = cls; return n; }
  function signIn() { location.assign('/login?next=' + encodeURIComponent(location.pathname + location.search)); }
  window.CampusLoopContact = function (listing) {
    const panel = document.querySelector('.campusloop-contact-panel');
    if (!panel || panel.dataset.chatReady === listing.id) return;
    panel.dataset.chatReady = listing.id;
    const sample = listing.seller_id === 'campusloop-demo-seller';
    panel.replaceChildren(el('summary', sample ? 'Chat preview' : 'Contact seller'));
    if (sample) {
      const log = el('div', '', 'campusloop-demo-chat'); log.setAttribute('role', 'log'); log.setAttribute('aria-live', 'polite');
      log.append(el('p', 'Reply: Hi! Ask about availability or campus pickup.'));
      const answer = text => {
        log.append(el('p', 'You: ' + text));
        let reply = 'You can ask about availability, pickup location, or timing.';
        if (/available|still/i.test(text)) reply = 'Yes, it is available.';
        else if (/where|pickup|pick up|location/i.test(text)) reply = 'Pickup is at ' + (listing.pickup || 'Snell Library lobby') + '.';
        else if (/when|time/i.test(text)) reply = 'Weekdays after 5pm work for pickup.';
        log.append(el('p', 'Reply: ' + reply));
        while (log.children.length > 30) log.firstChild.remove();
        log.scrollTop = log.scrollHeight;
      };
      const choices = el('div', '', 'campusloop-message-chips');
      for (const text of ['Is this still available?', 'Where can I pick this up?', 'What time works?']) { const b = el('button', text); b.type = 'button'; b.onclick = () => answer(text); choices.append(b); }
      const form = el('form', '', 'campusloop-chat-form'); const input = document.createElement('input'); input.placeholder = 'Write a message…'; input.setAttribute('aria-label', 'Message'); input.maxLength = 2000; input.required = true;
      const send = el('button', 'Send'); send.type = 'submit'; form.append(input, send);
      form.onsubmit = e => { e.preventDefault(); if (input.value.trim()) answer(input.value.trim()); input.value = ''; };
      panel.append(choices, log, form); return;
    }
    const user = backend.getUser();
    const own = user && user.id === listing.seller_id;
    panel.querySelector('summary').textContent = own ? 'Buyer messages' : 'Contact seller';
    panel.append(el('p', own ? 'This is your listing. Messages from interested buyers will appear in your inbox.' : 'Ask the seller about this item or arrange pickup.', 'campusloop-contact-help'));
    const button = el('button', user && user.id === listing.seller_id ? 'View buyer messages' : 'Message seller', 'campusloop-button campusloop-button-primary campusloop-contact-action'); button.type = 'button';
    const status = el('p', ''); status.setAttribute('role', 'status');
    button.onclick = async () => {
      if (!backend.getUser()) { signIn(); return; }
      if (backend.getUser().id === listing.seller_id) { location.assign('/messages'); return; }
      button.disabled = true; status.textContent = 'Opening conversation…';
      try { const c = await backend.api('/listings/' + encodeURIComponent(listing.id) + '/conversation', {method:'POST'}); location.assign('/messages?conversation=' + encodeURIComponent(c.id)); }
      catch (e) { status.textContent = e.message; button.disabled = false; }
    };
    panel.append(button, status);
  };
  async function inbox() {
    const target = document.getElementById('chat-conversations'); if (!target) return;
    const user = backend.getUser(); if (!user) { signIn(); return; }
    const status = document.getElementById('chat-status'), thread = document.getElementById('chat-thread'), log = document.getElementById('chat-messages'), form = document.getElementById('chat-form'), input = document.getElementById('chat-body');
    let selected = new URLSearchParams(location.search).get('conversation'), signature = '', listSignature = '', busy = false, pending = null;
    async function refresh() {
      if (busy || document.hidden) return; busy = true;
      try {
        const conversations = await backend.api('/conversations');
        const next = JSON.stringify(conversations);
        if (next !== listSignature) {
          target.replaceChildren();
          for (const c of conversations) { const b = el('button', c.title + ' · ' + c.other_name, 'campusloop-conversation'); b.type = 'button'; b.dataset.conversation = c.id; b.onclick = () => { selected = c.id; signature = ''; thread.hidden = true; log.replaceChildren(); input.value = ''; pending = null; history.replaceState(null, '', '/messages?conversation=' + encodeURIComponent(c.id)); refresh(); }; target.append(b); }
          listSignature = next;
        }
        const c = conversations.find(c => c.id === selected);
        status.textContent = conversations.length ? (c ? '' : 'Select a conversation.') : 'No conversations yet. Message another seller from their item page, or wait for a buyer to contact you about your listing.';
        thread.hidden = !c;
        if (c) {
          const requested = selected;
          document.getElementById('chat-title').textContent = c.other_name;
          const link = document.getElementById('chat-item'); link.href = '/product/' + encodeURIComponent(c.listing_id); link.textContent = c.title;
          for (const b of target.children) b.setAttribute('aria-pressed', String(b.dataset.conversation === selected));
          const messages = await backend.api('/conversations/' + encodeURIComponent(requested) + '/messages');
          if (selected !== requested) return;
          const state = JSON.stringify(messages);
          if (state !== signature) { log.replaceChildren(...messages.map(m => { const row = el('div', '', m.sender_id === user.id ? 'campusloop-message-own' : 'campusloop-message-other'); row.append(el('strong', m.sender_id === user.id ? 'You' : c.other_name), el('p', m.body), el('small', new Date(m.created_at).toLocaleString())); return row; })); signature = state; log.scrollTop = log.scrollHeight; }
        }
      } catch (e) { status.textContent = e.message; } finally { busy = false; }
    }
    form.onsubmit = async e => {
      e.preventDefault(); const body = input.value.trim(); if (!body || !selected) return;
      const conversation = selected, button = form.querySelector('button'); button.disabled = true; input.disabled = true;
      if (!pending || pending.body !== body || pending.conversation !== conversation) pending = {body, conversation, id:crypto.randomUUID()};
      status.textContent = 'Sending…';
      try { await backend.api('/conversations/' + encodeURIComponent(conversation) + '/messages', {method:'POST',body:{body,client_id:pending.id}}); if (selected === conversation) input.value = ''; pending = null; status.textContent = ''; await refresh(); }
      catch (e) { status.textContent = e.message + ' You can retry sending.'; }
      finally { button.disabled = false; input.disabled = false; }
    };
    await refresh();
    let stopped = false;
    async function poll() { if (stopped) return; await refresh(); if (!stopped) setTimeout(poll, 5000); }
    setTimeout(poll, 5000); window.addEventListener('pagehide', () => { stopped = true; });
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', inbox); else inbox();
}());
