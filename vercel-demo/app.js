const products = [
  {
    id: "used-calculus-textbook",
    name: "Used Calculus Textbook",
    price: 32.98,
    category: "textbooks",
    image: "/assets/products/used-calculus-textbook.png",
    description: "Clean used calculus textbook with tabs and light notes. Good for first-year math courses.",
    condition: "Good",
    pickup: "Snell Library lobby",
    handoff: "Weekdays after 5pm"
  },
  {
    id: "campus-club-hoodie",
    name: "Campus Club Hoodie",
    price: 24.99,
    category: "clothing",
    image: "/assets/products/campus-club-hoodie.png",
    description: "Pre-owned university hoodie in good condition. Cozy for late library nights and campus events.",
    condition: "Good",
    pickup: "Curry Student Center",
    handoff: "Weekdays after 5pm"
  },
  {
    id: "study-desk-monitor",
    name: "Used 24-inch Study Monitor",
    price: 85.98,
    category: "electronics",
    image: "/assets/products/study-desk-monitor.png",
    description: "24-inch desk monitor for studying, coding, or dorm setup. Includes power cable.",
    condition: "Good",
    pickup: "ISEC lobby",
    handoff: "Friday afternoon"
  },
  {
    id: "adjustable-desk-lamp",
    name: "Adjustable Desk Lamp",
    price: 18.5,
    category: "dorm",
    image: "/assets/products/adjustable-desk-lamp.png",
    description: "Black adjustable lamp with a compact base. Useful for dorm desks and night study.",
    condition: "Good",
    pickup: "LightView lobby",
    handoff: "Sunday afternoon"
  },
  {
    id: "mini-dorm-fan",
    name: "Mini Dorm Fan",
    price: 14.0,
    category: "dorm",
    image: "/assets/products/mini-dorm-fan.png",
    description: "Small white fan for dorm rooms. Quiet and easy to move.",
    condition: "Like new",
    pickup: "West Village entrance",
    handoff: "Evenings"
  },
  {
    id: "reusable-water-bottle",
    name: "Reusable Water Bottle",
    price: 9.5,
    category: "essentials",
    image: "/assets/products/reusable-water-bottle.png",
    description: "Insulated bottle with a carry handle. Lightly used and cleaned.",
    condition: "Good",
    pickup: "Marino Center",
    handoff: "Any weekday"
  },
  {
    id: "intro-cs-notes-bundle",
    name: "Intro CS Notes Bundle",
    price: 12.0,
    category: "textbooks",
    image: "/assets/products/intro-cs-notes-bundle.png",
    description: "Printed intro CS notes and practice sheets. Helpful for review before exams.",
    condition: "Good",
    pickup: "Snell Library lobby",
    handoff: "Tuesday or Thursday"
  },
  {
    id: "mechanical-keyboard",
    name: "Mechanical Keyboard",
    price: 39.0,
    category: "electronics",
    image: "/assets/products/mechanical-keyboard.png",
    description: "Compact mechanical keyboard for desk setup. Works well for coding and writing.",
    condition: "Good",
    pickup: "Ruggles Station entrance",
    handoff: "Weekends"
  },
  {
    id: "commuter-bike",
    name: "Campus Commuter Bike",
    price: 120.0,
    category: "transportation",
    image: "/assets/products/commuter-bike.png",
    description: "Simple commuter bike for campus trips. Best picked up near Ruggles.",
    condition: "Used",
    pickup: "Ruggles Station entrance",
    handoff: "Saturday morning"
  }
];

const categories = ["all", "textbooks", "furniture", "dorm", "electronics", "clothing", "essentials", "other"];
const app = document.getElementById("app");
const searchInput = document.getElementById("search-input");
const searchForm = document.getElementById("search-form");

function money(value) {
  return "$" + Number(value || 0).toFixed(2);
}

function getUser() {
  try {
    return JSON.parse(localStorage.getItem("campusloop-user")) || null;
  } catch {
    return null;
  }
}

function setUser(user) {
  localStorage.setItem("campusloop-user", JSON.stringify(user));
}

function getList(key) {
  try {
    return JSON.parse(localStorage.getItem(key)) || [];
  } catch {
    return [];
  }
}

function setList(key, value) {
  localStorage.setItem(key, JSON.stringify(value));
}

function initials(name, email) {
  const value = String(name || email || "Student").trim();
  const parts = value.split(/\s+/).filter(Boolean);
  if (parts.length > 1) return (parts[0][0] + parts[1][0]).toUpperCase();
  return value.slice(0, 2).toUpperCase();
}

function allProducts() {
  const userListings = getList("campusloop-my-listings").map((item) => ({
    id: item.id,
    name: item.name,
    price: Number(item.price || 0),
    category: item.category || "other",
    image: item.image || "/assets/products/campus-club-hoodie.png",
    description: item.description || "Student-posted CampusLoop listing.",
    condition: item.condition || "Good",
    pickup: item.pickup || "Pickup TBD",
    handoff: item.handoff || "Handoff time TBD",
    status: item.status || "Available",
    campus: item.campus || "Northeastern University - Boston"
  }));
  return [...userListings, ...products];
}

function findProduct(id) {
  return allProducts().find((item) => item.id === id) || products[0];
}

function navigate(path) {
  history.pushState({}, "", "/#" + path);
  render();
  window.scrollTo({ top: 0, behavior: "smooth" });
}

function updateHeader() {
  const user = getUser();
  const login = document.querySelector("[data-auth-login]");
  const menuLogin = document.querySelector("[data-menu-login]");
  const userMenu = document.querySelector("[data-auth-menu]");
  const avatar = document.querySelector("[data-auth-avatar]");
  const name = document.querySelector("[data-auth-name]");
  const role = document.querySelector("[data-auth-role]");
  const cartCount = document.querySelector("[data-cart-count]");
  const cart = getList("campusloop-cart");

  if (login) login.hidden = Boolean(user);
  if (menuLogin) menuLogin.textContent = user ? "Messages" : "Log in";
  if (menuLogin) menuLogin.setAttribute("href", user ? "/messages" : "/login");
  if (userMenu) userMenu.hidden = !user;
  if (user && avatar) avatar.textContent = initials(user.name, user.email);
  if (user && name) name.textContent = user.name || user.email;
  if (user && role) role.textContent = user.role === "seller" ? "Seller mode" : "Buyer mode";
  if (cartCount) {
    const count = cart.reduce((sum, item) => sum + Number(item.quantity || 1), 0);
    cartCount.textContent = String(count);
    cartCount.hidden = count === 0;
  }
}

function closeOpenMenus() {
  document.querySelectorAll("details[open]").forEach((detail) => detail.removeAttribute("open"));
}

function productCard(product) {
  return `
    <article class="card" data-card data-category="${product.category}">
      <a class="card-media" href="/product/${product.id}" data-link>
        <img src="${product.image}" alt="${product.name}">
        <span class="price-pill">${money(product.price)}</span>
        <span class="status-pill">${product.status || "Available"}</span>
      </a>
      <div class="card-body">
        <h3>${product.name}</h3>
        <div class="card-footer">
          <span>Northeastern</span>
          <span>View item</span>
        </div>
      </div>
    </article>
  `;
}

function renderHome() {
  const route = currentRoute();
  const params = new URLSearchParams(route.search);
  const active = params.get("category") || "all";
  const query = (params.get("q") || "").trim().toLowerCase();
  const user = getUser();
  const visibleProducts = allProducts().filter((product) => {
    const categoryMatch = active === "all" || product.category === active;
    const searchMatch = !query || [product.name, product.description, product.category, product.pickup].join(" ").toLowerCase().includes(query);
    return categoryMatch && searchMatch;
  });

  if (searchInput) searchInput.value = query;
  app.classList.remove("side-rail");
  app.innerHTML = `
    ${user ? "" : `
      <section class="hero">
        <div>
          <p class="eyebrow">Northeastern marketplace</p>
          <h1>Campus finds, student to student</h1>
          <p>Buy, sell, and pick up around Northeastern.</p>
          <div class="hero-actions">
            <a class="button primary" href="/login" data-link>Log in</a>
            <a class="button" href="/post-item" data-link>Sell</a>
          </div>
        </div>
      </section>
    `}
    <section class="section">
      <div class="section-header">
        <div>
          <h2>Recently Listed</h2>
          <p>Northeastern student listings near Boston campus.</p>
        </div>
        <div class="tabs">
          ${categories.map((category) => `<button class="${category === active ? "active" : ""}" data-category="${category}">${category === "all" ? "All" : label(category)}</button>`).join("")}
        </div>
      </div>
      <div class="grid">${visibleProducts.map(productCard).join("")}</div>
      ${visibleProducts.length ? "" : `<p class="empty-state">No listings match your search yet.</p>`}
    </section>
    <a class="ad-strip" href="/product/campus-club-hoodie" data-link>Campus Club Hoodie for sale. 20% off.</a>
  `;
}

function label(value) {
  return String(value || "other").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function renderProduct(id) {
  const product = findProduct(id);
  const saved = getList("campusloop-saved-items").some((item) => item.id === product.id);
  app.classList.remove("side-rail");
  app.innerHTML = `
    <section class="product-page">
      <img class="product-image" src="${product.image}" alt="${product.name}">
      <div class="product-info">
        <p class="eyebrow">Northeastern student listing</p>
        <button class="save-button ${saved ? "saved" : ""}" data-save="${product.id}">${saved ? "Saved" : "Save"}</button>
        <h1>${product.name}</h1>
        <p class="product-price">${money(product.price)}</p>
        <p>${product.description}</p>
        <div class="details-list">
          <div><span>Status</span><strong>${product.status || "Available"}</strong></div>
          <div><span>Condition</span><strong>${product.condition}</strong></div>
          <div><span>Pickup</span><strong>${product.pickup}</strong></div>
          <div><span>Handoff time</span><strong>${product.handoff}</strong></div>
        </div>
        <details class="panel">
          <summary>Message seller</summary>
          <div class="message-chips">
            <button data-prompt="Is this still available?">Is this still available?</button>
            <button data-prompt="Where can I pick this up?">Where can I pick this up?</button>
            <button data-prompt="What time works?">What time works?</button>
          </div>
          <div class="chat-window" data-chat-window hidden>
            <div class="chat-header">
              <strong>Message seller</strong>
              <button type="button" data-chat-close>Close</button>
            </div>
            <div class="chat-messages" data-chat-messages>
              <div class="bubble seller">Hi! Ask me about pickup or availability.</div>
            </div>
            <form class="chat-form" data-chat-form>
              <input name="message" placeholder="Type a message..." autocomplete="off">
              <button type="submit">Send</button>
            </form>
          </div>
        </details>
        <details class="panel">
          <summary>Pickup details</summary>
          <p>${product.campus || "Northeastern University - Boston"} · ${product.pickup} · Public spots preferred.</p>
        </details>
        <div class="hero-actions" style="justify-content:flex-start;margin-top:30px">
          <button class="button primary" data-add-cart="${product.id}">Add To Cart</button>
        </div>
      </div>
    </section>
    ${renderRecommendations(product.id)}
  `;
}

function renderRecommendations(currentId) {
  const recs = products.filter((product) => product.id !== currentId).slice(0, 4);
  return `
    <section class="recommendations section">
      <h2>You May Also Like</h2>
      <div class="rec-grid">
        ${recs.map((product) => `
          <a href="/product/${product.id}" data-link>
            <img src="${product.image}" alt="${product.name}">
            <strong>${product.name}</strong>
          </a>
        `).join("")}
      </div>
    </section>
  `;
}

function renderAuth(mode = "login") {
  app.classList.remove("side-rail");
  const isSignup = mode === "signup" || currentRoute().path === "/signup";
  app.innerHTML = `
    <section class="auth-card">
      <p class="eyebrow">CampusLoop account</p>
      <div class="auth-tabs">
        <button class="${isSignup ? "" : "active"}" data-auth-tab="login">Log in</button>
        <button class="${isSignup ? "active" : ""}" data-auth-tab="signup">Create account</button>
      </div>
      <form class="auth-form" data-auth-form>
        <label>Email<input type="email" name="email" placeholder="student@example.com" required></label>
        <label>Display name<input type="text" name="name" placeholder="Alex" required></label>
        <label class="check-row"><input type="checkbox" name="seller"><span>I also want to sell items</span></label>
        <button class="button primary" type="submit">${isSignup ? "Create account" : "Log in"}</button>
      </form>
    </section>
  `;
}

function requireUser(nextPath) {
  if (getUser()) return true;
  navigate(`/login?next=${encodeURIComponent(nextPath)}`);
  return false;
}

function renderPostItem() {
  if (!requireUser(currentRoute().path)) return;
  app.classList.add("side-rail");
  app.innerHTML = `
    <section class="dashboard-card">
      <div class="dashboard-header">
        <div>
          <p class="eyebrow">Student seller</p>
          <h1>Post an item</h1>
        </div>
        <a class="button small" href="/my-listings" data-link>My listings</a>
      </div>
      <form class="form-grid" data-post-form>
        <label class="wide">Item photo<input name="image" type="file" accept="image/*"></label>
        <label>Title<input name="name" placeholder="Desk lamp, calculus textbook, mini fridge..." required></label>
        <label>Category<select name="category">${["textbooks", "furniture", "dorm", "electronics", "clothing", "transportation", "essentials", "other"].map((cat) => `<option value="${cat}">${label(cat)}</option>`).join("")}</select></label>
        <label>Price<input name="price" type="number" min="0" step="1" placeholder="25" required></label>
        <label>Status<select name="status"><option>Available</option><option>Reserved</option><option>Sold</option></select></label>
        <label>City<input name="city" value="Boston" required></label>
        <label>Campus<select name="campus"><option>Northeastern University - Boston</option><option>Northeastern University - Oakland</option><option>Northeastern University - Seattle</option><option>Northeastern University - Vancouver</option></select></label>
        <label>Dorm area or neighborhood<input name="dorm" placeholder="LightView, IV, West Village, Mission Hill..." required></label>
        <label>Recommended pickup spot<select name="pickup"><option>Snell Library lobby</option><option>Curry Student Center entrance</option><option>ISEC lobby</option><option>Marino Recreation Center entrance</option><option>Ruggles Station entrance</option><option>LightView lobby</option></select></label>
        <label class="wide">Pickup address or notes<input name="address" placeholder="Meet inside lobby; exact dorm room shared after purchase" required></label>
        <label>Contact method<input name="contact" placeholder="Email, phone, Instagram..." required></label>
        <label>Available handoff time<input name="handoff" placeholder="Weekdays after 5pm, Sunday afternoon..." required></label>
        <label class="wide">Description<input name="description" placeholder="Condition, included accessories, original price, notes..." required></label>
        <button class="button primary" type="submit">Save listing</button>
      </form>
    </section>
  `;
}

function readImage(file) {
  return new Promise((resolve) => {
    if (!file) {
      resolve("");
      return;
    }
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ""));
    reader.onerror = () => resolve("");
    reader.readAsDataURL(file);
  });
}

function renderMessages() {
  if (!requireUser(currentRoute().path)) return;
  const messages = getList("campusloop-messages").slice().reverse();
  app.classList.add("side-rail");
  app.innerHTML = `
    <section class="dashboard-card">
      <div class="dashboard-header">
        <div>
          <p class="eyebrow">Messages</p>
          <h1>Seller chats</h1>
        </div>
        <a class="button small" href="/" data-link>Browse items</a>
      </div>
      <div class="list-stack">
        ${messages.length ? messages.map((message) => `
          <a class="list-row" href="/product/${message.productId}" data-link>
            <div>
              <strong>${message.product}</strong>
              <span>${message.text}</span>
              <small>${message.reply}</small>
            </div>
          </a>
        `).join("") : `<div class="empty-state">No messages yet.</div>`}
      </div>
    </section>
  `;
}

function renderMyListings() {
  if (!requireUser(currentRoute().path)) return;
  const listings = getList("campusloop-my-listings");
  app.classList.add("side-rail");
  app.innerHTML = `
    <section class="dashboard-card">
      <div class="dashboard-header">
        <div>
          <p class="eyebrow">Seller tools</p>
          <h1>My listings</h1>
        </div>
        <a class="button primary small" href="/post-item" data-link>Post item</a>
      </div>
      <div class="list-stack">
        ${listings.length ? listings.map((listing) => `
          <div class="list-row">
            <div>
              <strong>${listing.name}</strong>
              <span>${listing.status || "Available"} · ${label(listing.category)} · ${listing.pickup || "Pickup TBD"}</span>
              <small>${listing.campus || "Northeastern"} · ${listing.dorm || "Dorm area TBD"} · ${listing.handoff || "Handoff time TBD"}</small>
            </div>
            <b>${money(listing.price)}</b>
          </div>
        `).join("") : `<div class="empty-state">No listings yet.</div>`}
      </div>
    </section>
  `;
}

function renderCart() {
  app.classList.add("side-rail");
  const cart = getList("campusloop-cart");
  const items = cart.map((entry) => ({ ...findProduct(entry.id), quantity: entry.quantity || 1 }));
  const total = items.reduce((sum, item) => sum + item.price * item.quantity, 0);
  app.innerHTML = `
    <section class="dashboard-card">
      ${items.length ? `
        <div class="dashboard-header">
          <div>
            <p class="eyebrow">My cart</p>
            <h1>Campus pickup checkout</h1>
          </div>
          <button class="button small" data-empty-cart>Empty cart</button>
        </div>
        <div class="cart-layout">
          <div>
            ${items.map((item) => `
              <div class="cart-item">
                <img src="${item.image}" alt="${item.name}">
                <div>
                  <strong>${item.name}</strong>
                  <p>${item.pickup} · Quantity ${item.quantity}</p>
                </div>
                <b>${money(item.price * item.quantity)}</b>
              </div>
            `).join("")}
          </div>
          <aside class="summary-box">
            <div><span>Campus pickup</span><strong>$0.00</strong></div>
            <div><span>Total</span><strong>${money(total)}</strong></div>
            <button class="button primary" type="button">Pay and Reserve Pickup</button>
          </aside>
        </div>
      ` : `
        <div class="empty-state">
          <h1>Your shopping cart is empty!</h1>
          <p>Items you add to your shopping cart will appear here.</p>
          <a class="button primary" href="/" data-link>Continue Shopping</a>
        </div>
        ${renderRecommendations("")}
      `}
    </section>
  `;
}

function sellerReply(text, product) {
  const lower = text.toLowerCase();
  if (lower.includes("where")) return `Pickup works at ${product.pickup}.`;
  if (lower.includes("time")) return `${product.handoff} usually works best.`;
  return "Yes, it is still available right now.";
}

function handleClick(event) {
  const link = event.target.closest("[data-link]");
  if (link) {
    event.preventDefault();
    closeOpenMenus();
    navigate(link.getAttribute("href"));
    return;
  }

  const categoryButton = event.target.closest("[data-category]");
  if (categoryButton) {
    navigate(`/?category=${categoryButton.dataset.category}`);
    return;
  }

  const authTab = event.target.closest("[data-auth-tab]");
  if (authTab) {
    renderAuth(authTab.dataset.authTab);
    return;
  }

  const logout = event.target.closest("[data-logout]");
  if (logout) {
    localStorage.removeItem("campusloop-user");
    closeOpenMenus();
    navigate("/");
    return;
  }

  const save = event.target.closest("[data-save]");
  if (save) {
    const product = findProduct(save.dataset.save);
    const saved = getList("campusloop-saved-items");
    const index = saved.findIndex((item) => item.id === product.id);
    if (index >= 0) saved.splice(index, 1);
    else saved.push({ id: product.id, name: product.name });
    setList("campusloop-saved-items", saved);
    renderProduct(product.id);
    return;
  }

  const addCart = event.target.closest("[data-add-cart]");
  if (addCart) {
    const cart = getList("campusloop-cart");
    const existing = cart.find((item) => item.id === addCart.dataset.addCart);
    if (existing) existing.quantity += 1;
    else cart.push({ id: addCart.dataset.addCart, quantity: 1 });
    setList("campusloop-cart", cart);
    updateHeader();
    navigate("/cart");
    return;
  }

  const prompt = event.target.closest("[data-prompt]");
  if (prompt) {
    const id = currentRoute().path.split("/").pop();
    const product = findProduct(id);
    addChatMessage(prompt.dataset.prompt, "buyer");
    window.setTimeout(() => addChatMessage(sellerReply(prompt.dataset.prompt, product), "seller", product, prompt.dataset.prompt), 220);
    return;
  }

  const closeChat = event.target.closest("[data-chat-close]");
  if (closeChat) {
    const chat = document.querySelector("[data-chat-window]");
    if (chat) chat.hidden = true;
    return;
  }

  if (event.target.closest("[data-empty-cart]")) {
    setList("campusloop-cart", []);
    updateHeader();
    renderCart();
  }
}

function addChatMessage(text, owner, product, originalText) {
  const chat = document.querySelector("[data-chat-window]");
  const messages = document.querySelector("[data-chat-messages]");
  if (!chat || !messages) return;
  chat.hidden = false;
  const bubble = document.createElement("div");
  bubble.className = `bubble ${owner}`;
  bubble.textContent = text;
  messages.appendChild(bubble);
  messages.scrollTop = messages.scrollHeight;
  if (owner === "seller" && product) {
    const stored = getList("campusloop-messages");
    stored.push({
      product: product.name,
      productId: product.id,
      text: originalText || "",
      reply: text,
      createdAt: new Date().toISOString()
    });
    setList("campusloop-messages", stored);
  }
}

function handleSubmit(event) {
  const authForm = event.target.closest("[data-auth-form]");
  if (authForm) {
    event.preventDefault();
    const formData = new FormData(authForm);
    const email = String(formData.get("email") || "").trim();
    setUser({
      email,
      name: String(formData.get("name") || email.split("@")[0] || "Student").trim(),
      role: formData.get("seller") ? "seller" : "buyer"
    });
    const next = new URLSearchParams(currentRoute().search).get("next") || "/";
    navigate(next);
    return;
  }

  const postForm = event.target.closest("[data-post-form]");
  if (postForm) {
    event.preventDefault();
    saveListing(postForm);
    return;
  }

  const chatForm = event.target.closest("[data-chat-form]");
  if (chatForm) {
    event.preventDefault();
    const input = chatForm.querySelector("input");
    const text = input.value.trim();
    if (!text) return;
    const product = findProduct(currentRoute().path.split("/").pop());
    addChatMessage(text, "buyer");
    input.value = "";
    window.setTimeout(() => addChatMessage(sellerReply(text, product), "seller", product, text), 220);
  }
}

async function saveListing(form) {
  const formData = new FormData(form);
  const imageInput = form.elements.image;
  const image = await readImage(imageInput && imageInput.files ? imageInput.files[0] : null);
  const listing = {
    id: "local-" + Date.now(),
    name: String(formData.get("name") || "").trim(),
    category: String(formData.get("category") || "other"),
    price: Number(formData.get("price") || 0),
    status: String(formData.get("status") || "Available"),
    city: String(formData.get("city") || "Boston"),
    campus: String(formData.get("campus") || "Northeastern University - Boston"),
    dorm: String(formData.get("dorm") || ""),
    pickup: String(formData.get("pickup") || ""),
    address: String(formData.get("address") || ""),
    contact: String(formData.get("contact") || ""),
    handoff: String(formData.get("handoff") || ""),
    description: String(formData.get("description") || ""),
    image
  };
  const listings = getList("campusloop-my-listings");
  listings.unshift(listing);
  setList("campusloop-my-listings", listings);
  navigate("/my-listings");
}

function render() {
  updateHeader();
  const route = currentRoute();
  const path = route.path;
  if (path === "/" || path === "/index.html") renderHome();
  else if (path === "/login") renderAuth("login");
  else if (path === "/signup") renderAuth("signup");
  else if (path === "/post-item") renderPostItem();
  else if (path === "/messages") renderMessages();
  else if (path === "/my-listings") renderMyListings();
  else if (path === "/cart") renderCart();
  else if (path.startsWith("/product/")) renderProduct(decodeURIComponent(path.split("/").pop()));
  else renderHome();
  updateHeader();
}

function currentRoute() {
  const hash = location.hash.startsWith("#") ? location.hash.slice(1) : "";
  const raw = hash || "/";
  const queryIndex = raw.indexOf("?");
  if (queryIndex >= 0) {
    return {
      path: raw.slice(0, queryIndex) || "/",
      search: raw.slice(queryIndex + 1)
    };
  }
  return {
    path: raw || "/",
    search: ""
  };
}

document.addEventListener("click", handleClick);
document.addEventListener("submit", handleSubmit);
window.addEventListener("popstate", render);
window.addEventListener("hashchange", render);

searchForm.addEventListener("submit", (event) => {
  event.preventDefault();
  const query = searchInput.value.trim();
  navigate(query ? `/?q=${encodeURIComponent(query)}` : "/");
});

render();
