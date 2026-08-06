const DEFAULT_BACKEND_URL = "http://127.0.0.1:8080";

export const API_BASE_URL = (
  import.meta.env.VITE_BACKEND_URL ??
  DEFAULT_BACKEND_URL
).replace(/\/$/, "");

const websocketURL = new URL(API_BASE_URL);

websocketURL.protocol =
  websocketURL.protocol === "https:"
    ? "wss:"
    : "ws:";

export const WS_BASE_URL =
  websocketURL.toString().replace(/\/$/, "");