<script>
  import { playerRanking, itemTypes } from '../stores/gameStore.js';

  const qualityNames = {
    common: '普通',
    fine: '精良',
    rare: '稀有',
    legendary: '传说',
  };

  function getItemName(typeId) {
    return $itemTypes[typeId]?.name || typeId;
  }
</script>

<div class="player-bar">
  {#each $playerRanking as player, index}
    <div class="player-card {player.isBankrupt ? 'bankrupt' : ''}">
      <div class="player-rank">#{index + 1}</div>
      <div class="player-info">
        <div class="player-header">
          <span class="player-name">{player.name}</span>
          <span class="shop-name">{player.shopName}</span>
        </div>
        <div class="player-stats">
          <span class="gold">💰 {player.gold}</span>
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
</style>
