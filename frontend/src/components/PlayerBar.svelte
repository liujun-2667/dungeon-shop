<script>
  import { get } from 'svelte/store';
  import { playerRanking, itemTypes, getReputationLevel, getReputationLabel, getReputationDescription, getReputationClass, getAuctionReputationColor, getAuctionReputationTier } from '../stores/gameStore.js';

  const qualityNames = {
    common: '普通',
    fine: '精良',
    rare: '稀有',
    legendary: '传说',
  };

  function getItemName(typeId) {
    return get(itemTypes)[typeId]?.name || typeId;
  }

  function getReputationTooltip(reputation) {
    const level = getReputationLevel(reputation);
    const label = getReputationLabel(level);
    const desc = getReputationDescription(level);
    return `声望: ${reputation}\n等级: ${label}\n效果: ${desc}`;
  }

  function getReputationIcon(reputation) {
    if (reputation >= 5) return '⭐';
    if (reputation <= -5) return '💀';
    if (reputation > 0) return '👍';
    if (reputation < 0) return '👎';
    return '😐';
  }

  function getAuctionReputationTooltip(auctionReputation) {
    const tier = getAuctionReputationTier(auctionReputation);
    const feeRate = auctionReputation >= 80 ? '3%' : auctionReputation >= 60 ? '5%' : auctionReputation >= 40 ? '8%' : '禁止挂单';
    return `拍卖行信誉: ${auctionReputation}\n等级: ${tier}\n挂单费率: ${feeRate}`;
  }
</script>

<div class="player-bar">
  {#each $playerRanking as player, index}
    {#if player.reputation !== undefined}
      {@const repLevel = getReputationLevel(player.reputation)}
      {@const repLabel = getReputationLabel(repLevel)}
      <div class="player-card {player.isBankrupt ? 'bankrupt' : ''} {getReputationClass(repLevel)}">
        <div class="player-rank">#{index + 1}</div>
        <div class="player-info">
          <div class="player-header">
            <span class="player-name">{player.name}</span>
            <span class="shop-name">{player.shopName}</span>
            {#if repLevel !== 'normal'}
              <span class="shop-badge badge-{repLevel}">{repLabel}</span>
            {/if}
          </div>
          <div class="player-stats">
            <span class="gold">💰 {player.gold}</span>
            {#if player.frozenGold > 0}
              <span class="frozen">❄️ {player.frozenGold}</span>
            {/if}
            {#if player.frozenDeposit > 0}
              <span class="frozen-deposit">🔒 {player.frozenDeposit}</span>
            {/if}
            <span class="assets">📊 {player.assets}</span>
            <span class="reputation" title="{getReputationTooltip(player.reputation)}">
              {getReputationIcon(player.reputation)} {player.reputation}
            </span>
            <span class="auction-reputation" 
                  style="color: {getAuctionReputationColor(player.auctionReputation)}"
                  title="{getAuctionReputationTooltip(player.auctionReputation)}">
              🏛️ {player.auctionReputation}
            </span>
          </div>
          <div class="player-shelves">
            {#each player.shelves as shelf}
              {#if shelf.item}
                <div class="shelf-item quality-{shelf.item.quality}" title="{getItemName(shelf.item.typeId)} {qualityNames[shelf.item.quality]}">
                  {shelf.price}💰
                </div>
              {:else}
                <div class="shelf-empty">-</div>
              {/if}
            {/each}
          </div>
        </div>
        {#if player.isBankrupt}
          <div class="bankrupt-badge">破产</div>
        {/if}
      </div>
    {:else}
      <div class="player-card {player.isBankrupt ? 'bankrupt' : ''}">
        <div class="player-rank">#{index + 1}</div>
        <div class="player-info">
          <div class="player-header">
            <span class="player-name">{player.name}</span>
            <span class="shop-name">{player.shopName}</span>
          </div>
          <div class="player-stats">
            <span class="gold">💰 {player.gold}</span>
            {#if player.frozenGold > 0}
              <span class="frozen">❄️ {player.frozenGold}</span>
            {/if}
            <span class="assets">📊 {player.assets}</span>
          </div>
          <div class="player-shelves">
            {#each player.shelves as shelf}
              {#if shelf.item}
                <div class="shelf-item quality-{shelf.item.quality}" title="{getItemName(shelf.item.typeId)} {qualityNames[shelf.item.quality]}">
                  {shelf.price}💰
                </div>
              {:else}
                <div class="shelf-empty">-</div>
              {/if}
            {/each}
          </div>
        </div>
        {#if player.isBankrupt}
          <div class="bankrupt-badge">破产</div>
        {/if}
      </div>
    {/if}
  {/each}
</div>

<style>
  .player-bar {
    display: flex;
    gap: 10px;
    overflow-x: auto;
    padding-bottom: 10px;
  }

  .player-card {
    flex: 1;
    min-width: 200px;
    padding: 12px;
    background: var(--card-bg);
    border-radius: 10px;
    border: 1px solid var(--border);
    position: relative;
  }

  .player-card.bankrupt {
    opacity: 0.5;
  }

  .player-rank {
    position: absolute;
    top: 8px;
    right: 8px;
    font-size: 18px;
    font-weight: bold;
    color: var(--primary);
  }

  .player-header {
    margin-bottom: 8px;
  }

  .player-name {
    font-weight: 600;
    margin-right: 8px;
  }

  .shop-name {
    font-size: 12px;
    color: var(--gray);
  }

  .player-stats {
    display: flex;
    gap: 15px;
    margin-bottom: 8px;
    font-size: 14px;
  }

  .gold {
    color: var(--secondary);
    font-weight: 600;
  }

  .frozen {
    color: #60a5fa;
    font-weight: 600;
    font-size: 13px;
  }

  .assets {
    color: var(--success);
    font-weight: 600;
  }

  .player-shelves {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
  }

  .shelf-item {
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 10px;
    background: rgba(0, 0, 0, 0.3);
    font-weight: 600;
  }

  .shelf-empty {
    padding: 2px 6px;
    font-size: 10px;
    color: var(--gray);
  }

  .bankrupt-badge {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%) rotate(-15deg);
    background: var(--danger);
    color: white;
    padding: 8px 16px;
    border-radius: 4px;
    font-weight: bold;
    font-size: 18px;
  }

  .reputation {
    font-weight: 600;
    cursor: help;
    padding: 1px 6px;
    border-radius: 4px;
    background: rgba(139, 92, 246, 0.15);
  }

  .player-card.reputation-honest {
    border-color: #10b981;
    box-shadow: 0 0 12px rgba(16, 185, 129, 0.3);
  }

  .player-card.reputation-shady {
    border-color: #ef4444;
    box-shadow: 0 0 12px rgba(239, 68, 68, 0.25);
  }

  .shop-badge {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 10px;
    font-size: 11px;
    font-weight: 600;
    margin-left: 6px;
    vertical-align: middle;
  }

  .shop-badge.badge-honest {
    background: linear-gradient(135deg, #10b981, #059669);
    color: white;
  }

  .shop-badge.badge-shady {
    background: linear-gradient(135deg, #ef4444, #dc2626);
    color: white;
  }

  .frozen-deposit {
    color: #f59e0b;
    font-weight: 600;
    font-size: 13px;
  }

  .auction-reputation {
    font-weight: 600;
    cursor: help;
    padding: 1px 6px;
    border-radius: 4px;
    background: rgba(16, 185, 129, 0.1);
    font-size: 13px;
  }
</style>
