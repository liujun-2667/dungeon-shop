<script>
  import { createEventDispatcher } from 'svelte';
  import { currentPlayer } from '../stores/gameStore.js';

  const dispatch = createEventDispatcher();

  const adventurerPool = [
    { name: '艾伦', level: 1, hireCost: 50 },
    { name: '贝拉', level: 2, hireCost: 80 },
    { name: '卡洛斯', level: 3, hireCost: 110 },
    { name: '黛安娜', level: 4, hireCost: 140 },
    { name: '伊森', level: 5, hireCost: 170 },
    { name: '菲奥娜', level: 2, hireCost: 80 },
  ];

  function hire(adventurerIdx) {
    const adv = adventurerPool[adventurerIdx];
    if (adv.hireCost > $currentPlayer.gold) {
      alert('金币不足！');
      return;
    }
    dispatch('hireAdventurer', adventurerIdx);
  }

  function close() {
    dispatch('close');
  }

  $: currentCount = $currentPlayer?.adventurers?.length || 0;
  $: maxCount = $currentPlayer?.maxAdventurers || 3;
  $: playerGold = $currentPlayer?.gold || 0;
</script>

<div class="modal-overlay" on:click={close}>
  <div class="modal card" on:click|stopPropagation>
    <div class="modal-header">
      <h2>👤 雇佣冒险者</h2>
      <div class="info">
        <span>💰 {playerGold}</span>
        <span>{currentCount}/{maxCount}</span>
      </div>
      <button class="close-btn" on:click={close}>×</button>
    </div>
    
    <div class="modal-content">
      {#if currentCount >= maxCount}
        <p class="empty">已达到最大冒险者数量</p>
      {:else}
        <div class="adventurer-list">
          {#each adventurerPool as adv, idx}
            <div class="adventurer-card">
              <div class="adv-info">
                <div class="adv-name">{adv.name}</div>
                <div class="adv-level">Lv.{adv.level}</div>
              </div>
              <div class="adv-stats">
                <div>成功率: {adv.level * 20}% (1层)</div>
                <div>雇佣费: {adv.hireCost}💰</div>
              </div>
              <button
                class="btn btn-primary small"
                on:click={() => hire(idx)}
                disabled={adv.hireCost > playerGold}
              >
                雇佣
              </button>
            </div>
          {/each}
        </div>
      {/if}
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

  .info {
    display: flex;
    gap: 15px;
    font-size: 14px;
  }

  .info span:first-child {
    color: var(--secondary);
    font-weight: 600;
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

  .adventurer-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .adventurer-card {
    display: flex;
    align-items: center;
    gap: 15px;
    padding: 12px 15px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 8px;
  }

  .adv-info {
    flex: 1;
  }

  .adv-name {
    font-weight: 600;
  }

  .adv-level {
    font-size: 12px;
    color: var(--secondary);
  }

  .adv-stats {
    font-size: 12px;
    color: var(--gray);
    text-align: right;
  }

  .btn.small {
    padding: 6px 12px;
    font-size: 12px;
  }

  .empty {
    text-align: center;
    color: var(--gray);
    padding: 40px;
  }
</style>
