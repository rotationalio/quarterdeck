const csrfErrorHeader = document.querySelector(
  'meta[name="csrf-error-header"]',
)?.content;

document.body.addEventListener("htmx:configRequest", (e) => {
  // Ensure the accept type for all HTMX requests is HTML partials.
  e.detail.headers["Accept"] = "text/html";
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
      notyf.error("Error: " + getResponseErrorMessage(e.detail.xhr));
      break;
  }
});

// HTMX middleware failures are not guaranteed to return JSON; normalize all
// response shapes into a message that can be shown without throwing.
function getResponseErrorMessage(xhr) {
  const responseText = (xhr.responseText || "").trim();
  const csrfError = csrfErrorHeader
    ? xhr.getResponseHeader(csrfErrorHeader)
    : null;

  if (xhr.status === 403 && csrfError) {
    return "Request blocked by CSRF protection. Fully reload the page and try again. Contact support if it still fails.";
  }

  if (responseText) {
    try {
      const error = JSON.parse(responseText);
      return error?.error || error?.message || responseText;
    } catch (_) {
      // Some middleware errors intentionally return plain text instead of JSON.
      return responseText;
    }
  }

  return xhr.statusText || `Request failed (${xhr.status})`;
}
