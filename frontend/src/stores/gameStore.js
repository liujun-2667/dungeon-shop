import { writable, derived } from 'svelte/store';

export const currentUser = writable({
  playerId: localStorage.getItem('playerId') || '',
  playerName: localStorage.getItem('playerName') || '',
  shopName: localStorage.getItem('shopName') || '',
});

export const roomId = writable('');
export const room = writable(null);
export const itemTypes = writable({});
export const websocket = writable(null);
export const currentPhase = writable('purchase');
export const phaseEndTime = writable(0);
export const eventLog = writable([]);
export const gameResults = writable(null);
export const isConnected = writable(false);

export const currentPlayer = derived([room, currentUser], ([$room, $user]) => {
  if (!$room || !$user.playerId) return null;
  return $room.players[$user.playerId] || null;
});

export const playerRanking = derived(room, ($room) => {
  if (!$room) return [];
  const players = Object.values($room.players);
  return players
    .map(p => ({
      id: p.id,
      name: p.name,
      shopName: p.shopName,
      gold: p.gold,
      assets: calculateAssets(p),
      isBankrupt: p.isBankrupt,
    }))
    .sort((a, b) => b.assets - a.assets);
});

function calculateAssets(player) {
  let value = player.gold;
  
  for (const item of player.warehouse) {
    value += item.purchaseCost * 0.8;
  }
  
  for (const slot of player.shelves) {
    if (slot.item) {
      value += slot.item.purchaseCost * 0.8;
    }
  }
  
  value += player.upgradeInvestment * 0.5;
  
  return Math.floor(value);
}

export function addLog(message, type = 'info') {
  eventLog.update(logs => [
    { id: Date.now(), message, type, time: new Date().toLocaleTimeString() },
    ...logs.slice(0, 50)
  ]);
}

export function setUser(playerId, playerName, shopName) {
  currentUser.set({ playerId, playerName, shopName });
  localStorage.setItem('playerId', playerId);
  localStorage.setItem('playerName', playerName);
  localStorage.setItem('shopName', shopName);
}

export function clearUser() {
  currentUser.set({ playerId: '', playerName: '', shopName: '' });
  localStorage.removeItem('playerId');
  localStorage.removeItem('playerName');
  localStorage.removeItem('shopName');
}
