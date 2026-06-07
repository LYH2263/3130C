const API_BASE = import.meta.env.VITE_API_BASE || '/api';
const DEFAULT_TIMEOUT = 15000;

let onUnauthorized = null;

export function setUnauthorizedHandler(handler) {
  onUnauthorized = handler;
}

export class ApiError extends Error {
  constructor(message, status, payload) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.payload = payload;
  }
}

async function parseResponse(response) {
  const contentType = response.headers.get('content-type') || '';
  const isJSON = contentType.includes('application/json');
  const payload = isJSON ? await response.json() : null;

  if (!response.ok) {
    const message = payload?.message || `Request failed: ${response.status}`;
    throw new ApiError(message, response.status, payload);
  }
  return payload;
}

export async function apiRequest(
  path,
  { method = 'GET', token, body, isForm = false, timeout = DEFAULT_TIMEOUT, signal } = {}
) {
  const headers = {};
  if (!isForm) {
    headers['Content-Type'] = 'application/json';
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const abortController = new AbortController();
  const timeoutId = setTimeout(() => abortController.abort(), timeout);

  const combinedSignal = signal
    ? (() => {
        const combinedController = new AbortController();
        signal.addEventListener('abort', () => combinedController.abort());
        abortController.signal.addEventListener('abort', () => combinedController.abort());
        return combinedController.signal;
      })()
    : abortController.signal;

  try {
    const response = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: isForm ? body : body ? JSON.stringify(body) : undefined,
      signal: combinedSignal,
    });

    if (response.status === 401 || response.status === 403) {
      if (onUnauthorized) {
        onUnauthorized(response.status);
      }
    }

    return await parseResponse(response);
  } catch (error) {
    if (error.name === 'AbortError') {
      throw new ApiError('请求超时，请稍后重试', 0, null);
    }
    throw error;
  } finally {
    clearTimeout(timeoutId);
  }
}
