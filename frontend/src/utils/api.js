const API_BASE = '/api';

async function request(url, options = {}) {
  const response = await fetch(`${API_BASE}${url}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    ...options,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

export const api = {
  createRoom: (name, maxPlayers, playerName, shopName) =>
    request('/rooms', {
      method: 'POST',
      body: JSON.stringify({ name, maxPlayers, playerName, shopName }),
    }),

  joinRoom: (roomId, playerName, shopName) =>
    request(`/rooms/${roomId}/join`, {
      method: 'POST',
      body: JSON.stringify({ playerName, shopName }),
    }),

  listRooms: () => request('/rooms'),

  getRoom: (roomId) => request(`/rooms/${roomId}`),

  startGame: (roomId) =>
    request(`/rooms/${roomId}/start`, { method: 'POST' }),

  getItemTypes: () => request('/item-types'),
};

export function connectWebSocket(roomId, playerId) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${window.location.host}/ws/${roomId}?playerId=${playerId}`;
  return new WebSocket(wsUrl);
}

export function sendWS(ws, type, data) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type, data }));
    return true;
  }
  return false;
}
