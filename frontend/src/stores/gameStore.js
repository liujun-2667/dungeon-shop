import { writable, derived } from 'svelte/store';

const QUALITY_MULTIPLIER = {
  common: 1.0,
  fine: 1.5,
  rare: 2.5,
  legendary: 4.0,
};

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
export const bargainRequests = writable([]);

export const currentPlayer = derived([room, currentUser], ([$room, $user]) => {
  if (!$room || !$user.playerId) return null;
  return $room.players[$user.playerId] || null;
});

export const playerRanking = derived([room, itemTypes], ([$room, $itemTypes]) => {
  if (!$room) return [];
  const players = Object.values($room.players);
  return players
    .map(p => ({
      id: p.id,
      name: p.name,
      shopName: p.shopName,
      gold: p.gold,
      assets: calculateAssets(p, $itemTypes),
      isBankrupt: p.isBankrupt,
      reputation: p.reputation ?? 0,
      shelves: p.shelves || [],
    }))
    .sort((a, b) => b.assets - a.assets);
});

function calculateAssets(player, itemTypesMap) {
  let value = player.gold;
  const types = itemTypesMap || {};

  const evalItem = (item) => {
    const itemType = types[item.typeId];
    if (itemType && QUALITY_MULTIPLIER[item.quality] !== undefined) {
      const basePrice = itemType.basePrice;
      const qualityMult = QUALITY_MULTIPLIER[item.quality];
      return basePrice * qualityMult * 0.8;
    }
    return (item.purchaseCost || 0) * 0.8;
  };

  for (const item of player.warehouse) {
    value += evalItem(item);
  }

  for (const slot of player.shelves) {
    if (slot.item) {
      value += evalItem(slot.item);
    }
  }

  if (player.branchShops) {
    for (const branch of player.branchShops) {
      for (const slot of branch.shelves) {
        if (slot.item) {
          value += evalItem(slot.item);
        }
      }
    }
  }

  value += player.upgradeInvestment * 0.5;

  return Math.floor(value);
}

export function calculateFinalAssets(player, itemTypesMap) {
  return calculateAssets(player, itemTypesMap);
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

export const REPUTATION_HONEST_THRESHOLD = 5;
export const REPUTATION_SHADY_THRESHOLD = -5;

export function getReputationLevel(reputation) {
  if (reputation >= REPUTATION_HONEST_THRESHOLD) return 'honest';
  if (reputation <= REPUTATION_SHADY_THRESHOLD) return 'shady';
  return 'normal';
}

export function getReputationLabel(level) {
  switch (level) {
    case 'honest': return '信誉商铺';
    case 'shady': return '黑心店铺';
    default: return '普通店铺';
  }
}

export function getReputationDescription(level) {
  switch (level) {
    case 'honest': return 'NPC到店概率+30%，愿意多花10%预算';
    case 'shady': return 'NPC到店概率-40%，预算上限压低20%';
    default: return '无额外加成或惩罚';
  }
}

export function getReputationClass(level) {
  switch (level) {
    case 'honest': return 'reputation-honest';
    case 'shady': return 'reputation-shady';
    default: return 'reputation-normal';
  }
}

export function addBargainRequest(request) {
  bargainRequests.update(list => [...list, request]);
}

export function removeBargainRequest(bargainId) {
  bargainRequests.update(list => list.filter(b => b.id !== bargainId));
}

export function clearBargainRequests() {
  bargainRequests.set([]);
}
