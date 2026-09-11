document.body.addEventListener("htmx:configRequest", (e) => {
  // Ensure the accept type for all HTMX requests is HTML partials.
  e.detail.headers["Accept"] = "text/html";

  // Copy the public CSRF token into the namespaced header only for the
  // configured page origin. Never send it to an unrelated origin.
  const requestURL = new URL(e.detail.path, window.location.href);
  const cookieName = document.querySelector(
    'meta[name="csrf-token-cookie"]',
  )?.content;
  const csrfHeader = document.querySelector(
    'meta[name="csrf-header"]',
  )?.content;
  const csrfToken = cookieName ? getCookie(cookieName) : null;
  if (requestURL.origin === window.location.origin && csrfHeader && csrfToken) {
    e.detail.headers[csrfHeader] = csrfToken;
  }
});

// Initialize and set Notyf config to display toast notifications.
const notyf = new Notyf({
  duration: 5000,
  ripple: false,
});

// Ensure that all 500 errors redirect to the error page.
document.body.addEventListener("htmx:responseError", (e) => {
  switch (e.detail.xhr.status) {
    case 500:
      window.location.href = "/error";
      break;
    case 501:
      window.location.href = "/not-allowed";
      break;
    default:
      const error = JSON.parse(e.detail.xhr.responseText);
      notyf.error("Error: " + error?.error || "An unknown error occurred");
  }
});

function getCookie(name) {
  const nameEQ = name + "=";
  const cookies = document.cookie.split(";");

  for (let cookie of cookies) {
    // Remove leading whitespace
    while (cookie.charAt(0) === " ") {
      cookie = cookie.substring(1);
    }

    // If cookie starts with the desired name, return its value, less the name part
    if (cookie.indexOf(nameEQ) === 0) {
      return cookie.substring(nameEQ.length, cookie.length);
    }
  }

  // Cookie not found
  return null;
}
