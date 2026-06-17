<script>
  import { currentPhase, currentPlayer } from '../stores/gameStore.js';

  export let showWholesaler;
  export let showHire;
  export let showUpgrade;
  export let showSynthesis;

  const phaseDescriptions = {
    purchase: '从批发商采购商品，摆上货架并定价',
    business: 'NPC冒险者光顾，自动购买商品',
    explore: '派遣冒险者深入地牢采集材料',
  };

  $: canPurchase = $currentPhase === 'purchase' && !$currentPlayer?.isBankrupt;
  $: canExplore = $currentPhase === 'explore' && !$currentPlayer?.isBankrupt;
</script>

<div class="action-bar">
  <div class="phase-info">
    <span class="phase phase-{$currentPhase}">
      {$currentPhase === 'purchase' ? '🛒' : $currentPhase === 'business' ? '🏪' : '⚔️'}
      {phaseDescriptions[$currentPhase]}
    </span>
  </div>
  
  <div class="action-buttons">
    {#if canPurchase}
      <button class="btn btn-primary" on:click={() => showWholesaler = true}>
        🛒 进货
      </button>
      <button class="btn btn-secondary" on:click={() => showHire = true}>
        👤 雇佣
      </button>
      <button class="btn btn-success" on:click={() => showSynthesis = true}>
        ⚗️ 合成
      </button>
      <button class="btn btn-warning" on:click={() => showUpgrade = true}>
        🔧 升级
      </button>
    {:else if canExplore}
      <span class="hint">选择冒险者并派遣到地牢层数</span>
    {:else}
      <span class="hint">营业中，NPC正在购物...</span>
    {/if}
  </div>
</div>

<style>
  .action-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px 20px;
    background: var(--card-bg);
    border-radius: 12px;
    border: 1px solid var(--border);
    margin-top: 10px;
  }

  .phase-info {
    flex: 1;
  }

  .phase {
    font-weight: 600;
    font-size: 14px;
  }

  .action-buttons {
    display: flex;
    gap: 10px;
    align-items: center;
  }

  .hint {
    color: var(--gray);
    font-size: 14px;
  }

  .btn-warning {
    background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
    color: white;
    box-shadow: 0 4px 15px rgba(245, 158, 11, 0.4);
  }

  .btn-warning:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(245, 158, 11, 0.6);
  }
</style>
