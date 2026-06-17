<script>
  import { createEventDispatcher } from 'svelte';
  import { currentPlayer } from '../stores/gameStore.js';

  const dispatch = createEventDispatcher();

  const upgrades = [
    {
      type: 'shelf',
      name: '扩展货架',
      desc: '+1 货架位 (最多12个)',
      cost: 200,
      icon: '🏪',
      maxed: () => ($currentPlayer?.maxShelves || 6) >= 12,
    },
    {
      type: 'decorate',
      name: '店铺装修',
      desc: '+20% NPC到店概率',
      cost: 500,
      icon: '✨',
      maxed: () => false,
    },
    {
      type: 'warehouse',
      name: '扩建仓库',
      desc: '库存上限 +20 (最多40)',
      cost: 300,
      icon: '📦',
      maxed: () => ($currentPlayer?.warehouseCapacity || 20) >= 40,
    },
    {
      type: 'branch',
      name: '开设分店',
      desc: '额外4个独立货架位',
      cost: 1000,
      icon: '🏬',
      maxed: () => ($currentPlayer?.branchShops?.length || 0) >= 2,
    },
  ];

  function upgrade(upgradeType) {
    const upg = upgrades.find(u => u.type === upgradeType);
    if (upg.cost > $currentPlayer.gold) {
      alert('金币不足！');
      return;
    }
    dispatch('upgradeShop', upgradeType);
  }

  function close() {
    dispatch('close');
  }

  $: playerGold = $currentPlayer?.gold || 0;
</script>

<div class="modal-overlay" on:click={close}>
  <div class="modal card" on:click|stopPropagation>
    <div class="modal-header">
      <h2>🔧 商店升级</h2>
      <div class="gold-display">💰 {playerGold}</div>
      <button class="close-btn" on:click={close}>×</button>
    </div>
    
    <div class="modal-content">
      <div class="upgrade-list">
        {#each upgrades as upg}
          <div class="upgrade-card {upg.maxed() ? 'maxed' : ''}">
            <div class="upgrade-icon">{upg.icon}</div>
            <div class="upgrade-info">
              <div class="upgrade-name">{upg.name}</div>
              <div class="upgrade-desc">{upg.desc}</div>
            </div>
            <div class="upgrade-cost">{upg.cost}💰</div>
            <button
              class="btn btn-primary small"
              on:click={() => upgrade(upg.type)}
              disabled={upg.cost > playerGold || upg.maxed()}
            >
              {upg.maxed() ? '已满级' : '升级'}
            </button>
          </div>
        {/each}
      </div>
    </div>
  </div>
</div>

<style>
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    width: 90%;
    max-width: 500px;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    padding-bottom: 15px;
    border-bottom: 1px solid var(--border);
  }

  .modal-header h2 {
    color: var(--primary);
  }

  .gold-display {
    font-size: 18px;
    font-weight: bold;
    color: var(--secondary);
  }

  .close-btn {
    background: none;
    border: none;
    color: var(--gray);
    font-size: 24px;
    cursor: pointer;
    padding: 0 10px;
  }

  .close-btn:hover {
    color: var(--light);
  }

  .modal-content {
    flex: 1;
    overflow-y: auto;
  }

  .upgrade-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .upgrade-card {
    display: flex;
    align-items: center;
    gap: 15px;
    padding: 15px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 8px;
  }

  .upgrade-card.maxed {
    opacity: 0.5;
  }

  .upgrade-icon {
    font-size: 32px;
  }

  .upgrade-info {
    flex: 1;
  }

  .upgrade-name {
    font-weight: 600;
    margin-bottom: 4px;
  }

  .upgrade-desc {
    font-size: 12px;
    color: var(--gray);
  }

  .upgrade-cost {
    font-size: 14px;
    color: var(--secondary);
    font-weight: 600;
  }

  .btn.small {
    padding: 6px 12px;
    font-size: 12px;
  }
</style>
