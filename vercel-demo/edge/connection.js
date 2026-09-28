/* Retry only reads, never automatically replay a mutation. No background keep-alive. */
window.CampusLoopConnection = async function (path, accept) {
  const deadline = Date.now() + 90000;
  while (Date.now() < deadline) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 12000);
    try {
      const response = await fetch(path, {signal: controller.signal, cache: 'no-store'});
      const text = await response.text();
      if (response.ok && accept(text, response)) return {text, response};
      if (response.status === 404) throw Object.assign(new Error('This item or page could not be found.'), {permanent: true});
    } catch (error) {
      if (error.permanent) throw error;
    } finally { clearTimeout(timer); }
    await new Promise(resolve => setTimeout(resolve, 2500));
  }
  throw new Error('The marketplace is taking longer than expected. Please try again.');
};
