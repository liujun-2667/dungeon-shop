<script>
  import { createEventDispatcher } from 'svelte';
  import { currentPlayer, itemTypes } from '../stores/gameStore.js';

  const dispatch = createEventDispatcher();

  const qualityNames = {
    common: '普通',
    fine: '精良',
    rare: '稀有',
    legendary: '传说',
  };

  function getItemInfo(typeId) {
    return $itemTypes[typeId] || { name: typeId };
  }

  function canSynthesize(recipe) {
    const matCounts = {};
    for (const mat of recipe.materials) {
      matCounts[mat] = (matCounts[mat] || 0) + 1;
    }

    const warehouse = $currentPlayer?.warehouse || [];
    const haveCounts = {};
    for (const item of warehouse) {
      haveCounts[item.typeId] = (haveCounts[item.typeId] || 0) + 1;
    }

    for (const [mat, count] of Object.entries(matCounts)) {
      if ((haveCounts[mat] || 0) < count) {
        return false;
      }
    }
    return true;
  }

  function synthesize(recipeId) {
    dispatch('startSynthesis', recipeId);
  }

  function close() {
    dispatch('close');
  }

  $: recipes = $currentPlayer?.recipes || [];
  $: warehouse = $currentPlayer?.warehouse || [];
</script>

<div class="modal-overlay" on:click={close}>
  <div class="modal card" on:click|stopPropagation>
    <div class="modal-header">
      <h2>⚗️ 合成台</h2>
      <button class="close-btn" on:click={close}>×</button>
    </div>
    
    <div class="modal-content">
      {#if recipes.length === 0}
        <p class="empty">暂无合成配方</p>
      {:else}
        <div class="recipe-list">
          {#each recipes as recipe}
            <div class="recipe-card">
              <div class="recipe-header">
                <span class="recipe-name">{recipe.name}</span>
                <span class="recipe-output quality-{recipe.outputQuality}">
                  → {getItemInfo(recipe.outputItemType).name} ({qualityNames[recipe.outputQuality]})
                </span>
              </div>
              <div class="recipe-materials">
                <span class="label">需要材料:</span>
                {#each recipe.materials as mat}
                  <span class="mat">{getItemInfo(mat).name}</span>
                {/each}
              </div>
              <div class="recipe-time">⏱️ 合成时间: 1周</div>
              <button
                class="btn btn-success small"
                on:click={() => synthesize(recipe.id)}
                disabled={!canSynthesize(recipe)}
              >
                {canSynthesize(recipe) ? '开始合成' : '材料不足'}
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

  .recipe-list {
    display: flex;
    flex-direction: column;
    gap: 15px;
  }

  .recipe-card {
    padding: 15px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 8px;
    border-left: 4px solid var(--success);
  }

  .recipe-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
  }

  .recipe-name {
    font-weight: 600;
  }

  .recipe-output {
    font-size: 13px;
    font-weight: 600;
  }

  .recipe-materials {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    margin-bottom: 10px;
    font-size: 12px;
  }

  .label {
    color: var(--gray);
  }

  .mat {
    padding: 2px 8px;
    background: rgba(139, 92, 246, 0.3);
    border-radius: 4px;
  }

  .recipe-time {
    font-size: 12px;
    color: var(--gray);
    margin-bottom: 10px;
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
