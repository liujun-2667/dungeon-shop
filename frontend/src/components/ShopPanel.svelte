<script>
  import { currentPlayer, room, currentPhase, itemTypes } from '../stores/gameStore.js';

  export let showWholesaler;
  export let showHire;
  export let showSynthesis;
  export let handleDragStart;
  export let handleDrop;
  export let setPrice;
  export let removeItem;
  export let dispatchAdventurer;

  let selectedFloor = {};

  const qualityNames = {
    common: '普通',
    fine: '精良',
    rare: '稀有',
    legendary: '传说',
  };

  const categoryNames = {
    weapon: '武器',
    armor: '防具',
    consumable: '消耗品',
    material: '材料',
  };

  function getItemInfo(typeId) {
    return $itemTypes[typeId] || { name: typeId, category: 'unknown' };
  }

  function handleDragOver(e) {
    e.preventDefault();
  }

  function onPriceChange(shelf, e) {
    const price = parseInt(e.target.value) || 0;
    if ($currentPhase === 'purchase') {
      setPrice(shelf.id, price);
    }
  }

  function onDispatch(adventurer) {
    const floor = selectedFloor[adventurer.id] || 1;
    dispatchAdventurer(adventurer.id, floor);
  }

  $: warehouse = $currentPlayer?.warehouse || [];
  $: shelves = $currentPlayer?.shelves || [];
  $: adventurers = $currentPlayer?.adventurers || [];
  $: recipes = $currentPlayer?.recipes || [];
  $: branchShops = $currentPlayer?.branchShops || [];
  $: warehouseCapacity = $currentPlayer?.warehouseCapacity || 20;
  $: maxAdventurers = $currentPlayer?.maxAdventurers || 3;
</script>

<div class="shop-panel">
  <div class="shop-sections">
    <div class="section card">
      <div class="section-header">
        <h3>🏪 货架 ({shelves.length}/{$currentPlayer?.maxShelves || 6})</h3>
        {#if $currentPhase === 'purchase'}
          <span class="hint">拖拽仓库商品到货架</span>
        {/if}
      </div>
      <div class="shelves-grid">
        {#each shelves as shelf}
          <div
            class="shelf-slot {shelf.item ? 'filled' : 'empty'}"
            on:dragover={handleDragOver}
            on:drop={() => handleDrop(shelf)}
          >
            {#if shelf.item}
              <div class="shelf-item quality-{shelf.item.quality}">
                <div class="item-name">{getItemInfo(shelf.item.typeId).name}</div>
                <div class="item-quality">{qualityNames[shelf.item.quality]}</div>
                {#if $currentPhase === 'purchase'}
                  <div class="price-input">
                    <input
                      type="number"
                      value={shelf.price}
                      on:input={(e) => onPriceChange(shelf, e)}
                      min="1"
                    />
                    <span>💰</span>
                  </div>
                  <button class="btn btn-danger tiny" on:click={() => removeItem(shelf.id)}>下架</button>
                {:else}
                  <div class="price-display">售价: {shelf.price}💰</div>
                {/if}
              </div>
            {:else}
              <div class="shelf-placeholder">空位</div>
            {/if}
          </div>
        {/each}
      </div>

      {#if branchShops.length > 0}
        {#each branchShops as branch, bIdx}
          <div class="branch-shop">
            <h4>🏬 分店 #{bIdx + 1}</h4>
            <div class="shelves-grid small">
              {#each branch.shelves as shelf}
                <div
                  class="shelf-slot {shelf.item ? 'filled' : 'empty'}"
                  on:dragover={handleDragOver}
                  on:drop={() => handleDrop(shelf)}
                >
                  {#if shelf.item}
                    <div class="shelf-item quality-{shelf.item.quality}">
                      <div class="item-name small">{getItemInfo(shelf.item.typeId).name}</div>
                      <div class="price-display">{shelf.price}💰</div>
                    </div>
                  {:else}
                    <div class="shelf-placeholder small">空位</div>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/each}
      {/if}
    </div>

    <div class="section card">
      <div class="section-header">
        <h3>📦 仓库 ({warehouse.length}/{warehouseCapacity})</h3>
        {#if $currentPhase === 'purchase'}
          <button class="btn btn-primary small" on:click={() => showWholesaler = true}>
            🛒 进货
          </button>
        {/if}
      </div>
      <div class="warehouse-grid">
        {#if warehouse.length === 0}
          <p class="empty">仓库空空如也</p>
        {:else}
          {#each warehouse as item}
            <div
              class="warehouse-item quality-{item.quality} category-{getItemInfo(item.typeId).category}"
              draggable={$currentPhase === 'purchase'}
              on:dragstart={() => handleDragStart(item)}
              title="{getItemInfo(item.typeId).name} - {qualityNames[item.quality]}"
            >
              <div class="item-icon">{getCategoryIcon(getItemInfo(item.typeId).category)}</div>
              <div class="item-details">
                <div class="item-name">{getItemInfo(item.typeId).name}</div>
                <div class="item-meta">
                  <span class="quality">{qualityNames[item.quality]}</span>
                  {#if item.expiresWeek > 0}
                    <span class="expiry">第{item.expiresWeek}周过期</span>
                  {/if}
                </div>
              </div>
              <div class="item-cost">成本:{item.purchaseCost}</div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <div class="section card">
      <div class="section-header">
        <h3>⚔️ 冒险者 ({adventurers.length}/{maxAdventurers})</h3>
        {#if $currentPhase === 'purchase'}
          <button class="btn btn-secondary small" on:click={() => showHire = true}>
            👤 雇佣
          </button>
        {/if}
      </div>
      <div class="adventurers-list">
        {#if adventurers.length === 0}
          <p class="empty">还没有雇佣冒险者</p>
        {:else}
          {#each adventurers as adv}
            <div class="adventurer-card {adv.isOnMission ? 'on-mission' : ''} {adv.isInjured ? 'injured' : ''}">
              <div class="adv-info">
                <div class="adv-name">{adv.name}</div>
                <div class="adv-level">Lv.{adv.level}</div>
              </div>
              <div class="adv-status">
                {#if adv.isOnMission}
                  <span class="status on-mission">探索中</span>
                {:else if adv.isInjured}
                  <span class="status injured">受伤(第{adv.injuredUntilWeek}周恢复)</span>
                {:else if $currentPhase === 'explore'}
                  <div class="dispatch-controls">
                    <select bind:value={selectedFloor[adv.id]}>
                      <option value={1}>第1层</option>
                      <option value={2}>第2层</option>
                      <option value={3}>第3层</option>
                      <option value={4}>第4层</option>
                      <option value={5}>第5层</option>
                    </select>
                    <button class="btn btn-success tiny" on:click={() => onDispatch(adv)}>
                      派遣
                    </button>
                  </div>
                {:else}
                  <span class="status available">空闲</span>
                {/if}
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <div class="section card">
      <div class="section-header">
        <h3>🔮 合成台 ({recipes.length}个配方)</h3>
        {#if $currentPhase === 'purchase' && recipes.length > 0}
          <button class="btn btn-success small" on:click={() => showSynthesis = true}>
            ⚗️ 合成
          </button>
        {/if}
      </div>
      <div class="recipes-list">
        {#if recipes.length === 0}
          <p class="empty">暂无配方，地牢探险可能获得</p>
        {:else}
          {#each recipes as recipe}
            <div class="recipe-card">
              <div class="recipe-name">{recipe.name}</div>
              <div class="recipe-materials">
                {#each recipe.materials as mat}
                  <span class="mat">{getItemInfo(mat).name}</span>
                {/each}
              </div>
              <div class="recipe-output">
                → {getItemInfo(recipe.outputItemType).name} ({qualityNames[recipe.outputQuality]})
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  </div>
</div>

<script context="module">
  function getCategoryIcon(category) {
    const icons = {
      weapon: '⚔️',
      armor: '🛡️',
      consumable: '🧪',
      material: '💎',
    };
    return icons[category] || '📦';
  }
</script>

<style>
  .shop-panel {
    height: 100%;
  }

  .shop-sections {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }

  .section {
    padding: 15px;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 15px;
  }

  .section-header h3 {
    color: var(--primary);
    font-size: 16px;
  }

  .hint {
    font-size: 12px;
    color: var(--gray);
  }

  .shelves-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }

  .shelves-grid.small {
    grid-template-columns: repeat(4, 1fr);
  }

  .shelf-slot {
    aspect-ratio: 1;
    border: 2px dashed var(--border);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.3s ease;
    min-height: 100px;
  }

  .shelf-slot.empty:hover {
    border-color: var(--primary);
    background: rgba(139, 92, 246, 0.1);
  }

  .shelf-slot.filled {
    border-style: solid;
  }

  .shelf-placeholder {
    color: var(--gray);
    font-size: 12px;
  }

  .shelf-placeholder.small {
    font-size: 10px;
  }

  .shelf-item {
    width: 100%;
    height: 100%;
    padding: 8px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    gap: 4px;
  }

  .item-name {
    font-weight: 600;
    font-size: 12px;
  }

  .item-name.small {
    font-size: 10px;
  }

  .item-quality {
    font-size: 10px;
    opacity: 0.8;
  }

  .price-input {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .price-input input {
    width: 50px;
    padding: 4px;
    font-size: 11px;
    text-align: center;
  }

  .price-display {
    font-size: 11px;
    color: var(--secondary);
    font-weight: 600;
  }

  .btn.tiny {
    padding: 4px 8px;
    font-size: 10px;
  }

  .branch-shop {
    margin-top: 15px;
    padding-top: 15px;
    border-top: 1px solid var(--border);
  }

  .branch-shop h4 {
    margin-bottom: 10px;
    color: var(--secondary);
  }

  .warehouse-grid {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-height: 300px;
    overflow-y: auto;
  }

  .warehouse-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 10px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 6px;
    cursor: grab;
    transition: all 0.2s ease;
  }

  .warehouse-item:hover {
    background: rgba(139, 92, 246, 0.2);
  }

  .warehouse-item:active {
    cursor: grabbing;
  }

  .item-icon {
    font-size: 20px;
  }

  .item-details {
    flex: 1;
  }

  .item-meta {
    display: flex;
    gap: 8px;
    font-size: 10px;
    color: var(--gray);
  }

  .expiry {
    color: var(--danger);
  }

  .item-cost {
    font-size: 11px;
    color: var(--secondary);
  }

  .empty {
    text-align: center;
    color: var(--gray);
    padding: 20px;
  }

  .adventurers-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .adventurer-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 12px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 6px;
  }

  .adventurer-card.injured {
    opacity: 0.6;
  }

  .adventurer-card.on-mission {
    border-left: 3px solid var(--secondary);
  }

  .adv-name {
    font-weight: 600;
  }

  .adv-level {
    font-size: 12px;
    color: var(--secondary);
  }

  .status {
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 4px;
  }

  .status.available {
    background: rgba(16, 185, 129, 0.2);
    color: var(--success);
  }

  .status.on-mission {
    background: rgba(245, 158, 11, 0.2);
    color: var(--secondary);
  }

  .status.injured {
    background: rgba(239, 68, 68, 0.2);
    color: var(--danger);
  }

  .dispatch-controls {
    display: flex;
    gap: 6px;
  }

  .dispatch-controls select {
    padding: 4px 8px;
    font-size: 11px;
  }

  .recipes-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 200px;
    overflow-y: auto;
  }

  .recipe-card {
    padding: 10px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 6px;
    font-size: 12px;
  }

  .recipe-name {
    font-weight: 600;
    margin-bottom: 4px;
    color: var(--secondary);
  }

  .recipe-materials {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-bottom: 4px;
  }

  .mat {
    padding: 2px 6px;
    background: rgba(139, 92, 246, 0.3);
    border-radius: 4px;
    font-size: 10px;
  }

  .recipe-output {
    color: var(--success);
    font-size: 11px;
  }

  .btn.small {
    padding: 6px 12px;
    font-size: 12px;
  }

  @media (max-width: 768px) {
    .shop-sections {
      grid-template-columns: 1fr;
    }
  }
</style>
