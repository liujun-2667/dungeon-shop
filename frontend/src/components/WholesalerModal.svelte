<script>
  import { createEventDispatcher } from 'svelte';
  import { room, itemTypes, currentPlayer } from '../stores/gameStore.js';

  const dispatch = createEventDispatcher();

  let quantities = {};

  const qualityNames = {
    common: '普通',
    fine: '精良',
    rare: '稀有',
    legendary: '传说',
  };

  function getItemInfo(typeId) {
    return $itemTypes[typeId] || { name: typeId };
  }

  function getQuantity(item) {
    const key = `${item.typeId}-${item.quality}`;
    return quantities[key] || 1;
  }

  function setQuantity(item, value) {
    const key = `${item.typeId}-${item.quality}`;
    const qty = Math.max(1, Math.min(item.quantity, parseInt(value) || 1));
    quantities[key] = qty;
  }

  function buy(item) {
    const qty = getQuantity(item);
    const totalCost = item.price * qty;
    if (totalCost > $currentPlayer.gold) {
      alert('金币不足！');
      return;
    }
    dispatch('buyItem', { itemTypeId: item.typeId, quality: item.quality, quantity: qty });
  }

  function close() {
    dispatch('close');
  }

  $: stock = $room?.wholesalerStock || [];
  $: playerGold = $currentPlayer?.gold || 0;
</script>

<div class="modal-overlay" on:click={close}>
  <div class="modal card" on:click|stopPropagation>
    <div class="modal-header">
      <h2>🛒 批发商</h2>
      <div class="gold-display">💰 {playerGold}</div>
      <button class="close-btn" on:click={close}>×</button>
    </div>
    
    <div class="modal-content">
      {#if stock.length === 0}
        <p class="empty">本周批发商已售罄</p>
      {:else}
        <div class="stock-list">
          {#each stock as item}
            <div class="stock-item quality-{item.quality}">
              <div class="item-info">
                <span class="item-name">{getItemInfo(item.typeId).name}</span>
                <span class="item-quality">{qualityNames[item.quality]}</span>
                <span class="item-stock">库存: {item.quantity}</span>
              </div>
              <div class="item-price">单价: {item.price}💰</div>
              <div class="buy-controls">
                <input
                  type="number"
                  min="1"
                  max={item.quantity}
                  value={getQuantity(item)}
                  on:input={(e) => setQuantity(item, e.target.value)}
                />
                <button
                  class="btn btn-primary small"
                  on:click={() => buy(item)}
                  disabled={item.price * getQuantity(item) > playerGold}
                >
                  购买 ({item.price * getQuantity(item)}💰)
                </button>
              </div>
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
    max-width: 600px;
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

  .stock-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .stock-item {
    display: flex;
    align-items: center;
    gap: 15px;
    padding: 12px 15px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 8px;
    border-left: 4px solid var(--primary);
  }

  .item-info {
    flex: 1;
  }

  .item-name {
    font-weight: 600;
    margin-right: 10px;
  }

  .item-quality {
    font-size: 12px;
    margin-right: 10px;
  }

  .item-stock {
    font-size: 12px;
    color: var(--gray);
  }

  .item-price {
    font-size: 14px;
    color: var(--secondary);
    font-weight: 600;
  }

  .buy-controls {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .buy-controls input {
    width: 60px;
    padding: 6px;
    text-align: center;
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
