(function () {
  'use strict';
  const redirects = { '/cart': '/my-listings', '/marketplace': '/' };
  if (redirects[location.pathname]) {
    location.replace(redirects[location.pathname] + location.search);
    return;
  }
  const retry = document.getElementById('connection-retry');
  const message = document.getElementById('connection-message');
  async function load() {
    retry.hidden = true;
    message.textContent = document.getElementById('preview-title') ? '' : 'This may take about a minute. Your page will open automatically when ready.';
    try {
      const result = await CampusLoopConnection('/_pages' + location.pathname + location.search, (text, response) => (response.headers.get('content-type') || '').includes('text/html') && text.includes('window.CampusLoopBackendConfig'));
      // This is our own server-rendered HTML, including its original scripts and escaped data.
      // Keep the public URL so forms, sessions and redirects retain their original behavior.
      document.open(); document.write(result.text); document.close();
    } catch (error) { message.textContent = error.message; retry.hidden = false; }
  }
  retry.onclick = load; load();
}());
