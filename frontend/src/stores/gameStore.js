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
export const auctionErrors = writable([]);
export const guilds = writable([]);
export const guildErrors = writable([]);

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
      frozenGold: p.frozenGold || 0,
      frozenDeposit: p.frozenDeposit || 0,
      assets: calculateAssets(p, $itemTypes),
      isBankrupt: p.isBankrupt,
      reputation: p.reputation ?? 0,
      auctionReputation: p.auctionReputation ?? 100,
      shelves: p.shelves || [],
    }))
    .sort((a, b) => b.assets - a.assets);
});

function calculateAssets(player, itemTypesMap) {
  let value = player.gold + (player.frozenGold || 0);
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

let logIdCounter = 0;

export function addLog(message, type = 'info') {
  logIdCounter++;
  eventLog.update(logs => {
    const newLogs = [...logs, { id: logIdCounter, message, type, time: new Date().toLocaleTimeString() }];
    return newLogs.slice(-50);
  });
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

export function addAuctionError(action, error) {
  auctionErrors.update(list => [...list, { id: Date.now(), action, error }]);
  setTimeout(() => {
    auctionErrors.update(list => list.slice(1));
  }, 3000);
}

export function getActiveAuctions(room) {
  if (!room || !room.auctions) return [];
  return room.auctions.filter(a => a.status === 'active');
}

export function getMyAuctions(room, playerId) {
  if (!room || !room.auctions) return [];
  return room.auctions.filter(a => a.sellerId === playerId);
}

export function getMyBids(room, playerId) {
  if (!room || !room.auctions) return [];
  return room.auctions.filter(a =>
    a.status === 'active' && a.bidHistory && a.bidHistory.some(b => b.bidderId === playerId)
  );
}

export function getMinBid(auction) {
  if (!auction) return 0;
  if (auction.currentPrice === auction.startingPrice && (!auction.bidHistory || auction.bidHistory.length === 0)) {
    return auction.startingPrice;
  }
  return Math.floor(auction.currentPrice * 1.1);
}

export function getRemainingWeeks(auction, currentWeek) {
  if (!auction) return 0;
  return Math.max(0, auction.endWeek - currentWeek);
}

export function getAuctionStatusText(status) {
  switch (status) {
    case 'active': return '竞拍中';
    case 'sold': return '已成交';
    case 'expired': return '已流拍';
    case 'cancelled': return '已取消';
    default: return status;
  }
}

export const AUCTION_REP_INITIAL = 100;
export const AUCTION_REP_HONEST_MIN = 80;
export const AUCTION_REP_NORMAL_MIN = 60;
export const AUCTION_REP_FORBID_MIN = 40;

export function getAuctionReputationColor(reputation) {
  if (reputation >= AUCTION_REP_HONEST_MIN) return '#10b981';
  if (reputation >= AUCTION_REP_NORMAL_MIN) return '#ffffff';
  return '#ef4444';
}

export function getAuctionReputationTier(reputation) {
  if (reputation >= AUCTION_REP_HONEST_MIN) return '信誉极佳';
  if (reputation >= AUCTION_REP_NORMAL_MIN) return '信誉良好';
  if (reputation >= AUCTION_REP_FORBID_MIN) return '信誉较差';
  return '信誉极差';
}

export function getAuctionListingFeeRate(reputation) {
  if (reputation >= AUCTION_REP_HONEST_MIN) return 0.03;
  if (reputation >= AUCTION_REP_NORMAL_MIN) return 0.05;
  return 0.08;
}

export function getAuctionListingFeeTier(reputation) {
  if (reputation >= AUCTION_REP_HONEST_MIN) return 'honest';
  if (reputation >= AUCTION_REP_NORMAL_MIN) return 'normal';
  if (reputation >= AUCTION_REP_FORBID_MIN) return 'shady';
  return 'forbid';
}

export function canListAuction(reputation) {
  return reputation >= AUCTION_REP_FORBID_MIN;
}

export function getPlayerAuctionReputation(room, playerId) {
  if (!room || !room.players || !playerId) return 100;
  const player = room.players[playerId];
  return player?.auctionReputation ?? 100;
}

export function getMyAuctionHistory(room, playerId) {
  if (!room || !room.auctions || !playerId) return [];
  return room.auctions
    .filter(a => a.sellerId === playerId && a.status !== 'active')
    .sort((a, b) => b.createdAt - a.createdAt)
    .slice(0, 20)
    .map(a => {
      const buyer = a.status === 'sold' ? a.highestBidderName : '无人竞拍';
      let repChange = 0;
      if (a.status === 'sold') repChange = 2;
      else if (a.status === 'expired') repChange = -3;
      else if (a.status === 'cancelled') repChange = -1;
      return {
        id: a.id,
        itemTypeName: a.itemTypeName,
        itemQuality: a.item?.quality,
        finalPrice: a.currentPrice,
        buyer,
        repChange,
        status: a.status,
        createdAt: a.createdAt,
      };
    });
}

export function getMyBidHistory(room, playerId) {
  if (!room || !room.auctions || !playerId) return [];
  return room.auctions
    .filter(a => a.status !== 'active' && a.bidHistory && a.bidHistory.some(b => b.bidderId === playerId))
    .sort((a, b) => b.createdAt - a.createdAt)
    .slice(0, 20)
    .map(a => {
      const myBids = a.bidHistory.filter(b => b.bidderId === playerId);
      const myMaxBid = myBids.length > 0 ? Math.max(...myBids.map(b => b.amount)) : 0;
      const won = a.status === 'sold' && a.highestBidderId === playerId;
      return {
        id: a.id,
        itemTypeName: a.itemTypeName,
        itemQuality: a.item?.quality,
        myMaxBid,
        finalPrice: a.currentPrice,
        won,
        status: a.status,
        createdAt: a.createdAt,
      };
    });
}

export const GUILD_MAX_LEVEL = 5;
export const GUILD_WAREHOUSE_PER_LEVEL = 5;
export const GUILD_UPGRADE_REQUIREMENTS = {
  2: 200,
  3: 500,
  4: 1000,
  5: 2000,
};

export function getCurrentGuild(guildsList, playerGuildId) {
  if (!guildsList || !playerGuildId) return null;
  return guildsList.find(g => g.id === playerGuildId) || null;
}

export function getGuildWarehouseCapacity(guildLevel) {
  return (guildLevel || 1) * GUILD_WAREHOUSE_PER_LEVEL;
}

export function getGuildUpgradeCost(currentLevel) {
  return GUILD_UPGRADE_REQUIREMENTS[currentLevel + 1] || 0;
}

export function getGuildMaxMembers(roomMaxPlayers) {
  return Math.ceil(roomMaxPlayers / 2);
}

export function getActiveGuildAuctions(room, guildId) {
  if (!room || !room.auctions || !guildId) return [];
  return room.auctions.filter(a => a.isGuildAuction && a.guildId === guildId && a.status === 'active');
}

export function addGuildError(action, error) {
  guildErrors.update(list => [...list, { id: Date.now(), action, error }]);
  setTimeout(() => {
    guildErrors.update(list => list.slice(1));
  }, 3000);
}

export function getPlayerGuildTag(room, playerId) {
  if (!room || !room.players || !playerId) return null;
  const player = room.players[playerId];
  if (!player || !player.guildId || !room.guilds) return null;
  const guild = room.guilds[player.guildId];
  return guild ? guild.abbreviation : null;
}

export function isGuildLeader(guild, playerId) {
  if (!guild || !playerId) return false;
  return guild.leaderId === playerId;
}
