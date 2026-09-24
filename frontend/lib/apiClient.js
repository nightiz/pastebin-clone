class BackendConnectionError extends Error {
  constructor(message = 'Cannot reach backend service. Make sure the Go server is running.') {
    super(message);
    this.name = 'BackendConnectionError';
    this.isConnectionError = true;
  }
}

class BackendAPIError extends Error {
  constructor(status, code, message) {
    super(message || 'An API error occurred');
    this.name = 'BackendAPIError';
    this.status = status;
    this.code = code || 'INTERNAL';
  }
}

function getBaseUrl() {
  const url = process.env.BACKEND_API_URL || 'http://localhost:8080';
  return url.replace(/\/+$/, '');
}

async function request(path, options = {}) {
  const url = `${getBaseUrl()}${path}`;
  let response;

  try {
    response = await fetch(url, options);
  } catch (err) {
    throw new BackendConnectionError(
      `Cannot connect to backend at ${getBaseUrl()}: ${err.message}`
    );
  }

  // Handle raw text endpoint
  if (options.asRaw) {
    if (!response.ok) {
      const text = await response.text();
      let code = 'NOT_FOUND';
      let message = 'Paste not found';
      try {
        const parsed = JSON.parse(text);
        if (parsed.error) {
          code = parsed.error.code;
          message = parsed.error.message;
        }
      } catch {
        // Not JSON
      }
      throw new BackendAPIError(response.status, code, message);
    }
    return await response.text();
  }

  let data;
  try {
    data = await response.json();
  } catch {
    data = null;
  }

  if (!response.ok) {
    const code = data?.error?.code || 'UNKNOWN_ERROR';
    const message = data?.error?.message || `Backend responded with HTTP ${response.status}`;
    throw new BackendAPIError(response.status, code, message);
  }

  return data;
}

const apiClient = {
  BackendConnectionError,
  BackendAPIError,

  async createPaste({ content, title, language, expiresInSeconds, burnAfterRead }) {
    const payload = {
      content,
      title: title || '',
      language: language || '',
      burn_after_read: Boolean(burnAfterRead),
    };

    if (expiresInSeconds !== undefined && expiresInSeconds !== null && expiresInSeconds !== '') {
      const exp = parseInt(expiresInSeconds, 10);
      if (!isNaN(exp)) {
        payload.expires_in_seconds = exp;
      }
    }

    return await request('/api/pastes', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  },

  async getPaste(id) {
    return await request(`/api/pastes/${encodeURIComponent(id)}`);
  },

  async getPasteRaw(id) {
    return await request(`/api/pastes/${encodeURIComponent(id)}/raw`, { asRaw: true });
  },

  async deletePaste(id, editToken) {
    return await request(`/api/pastes/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: {
        'X-Edit-Token': editToken || '',
      },
    });
  },

  async checkHealth() {
    try {
      await request('/healthz');
      return true;
    } catch {
      return false;
    }
  }
};

module.exports = apiClient;
