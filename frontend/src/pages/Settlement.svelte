<script>
  import { onMount } from 'svelte';
  import { derived } from 'svelte/store';
  import { navigate } from 'svelte-routing';
  import { Line, Pie } from 'svelte-chartjs';
  import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Tooltip,
    Legend,
    ArcElement,
  } from 'chart.js';
  import { currentUser, currentPlayer, room } from '../stores/gameStore.js';

  ChartJS.register(
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Tooltip,
    Legend,
    ArcElement
  );

  let results = null;
  let myResult = null;

  const lineChartData = derived(currentPlayer, ($player) => {
    if (!$player) return null;
    
    const labels = $player.assetHistory?.map((_, i) => `第${i}周`) || [];
    
    return {
      labels,
      datasets: [
        {
          label: '资产总值',
          data: $player.assetHistory || [],
          borderColor: '#8b5cf6',
          backgroundColor: 'rgba(139, 92, 246, 0.1)',
          fill: true,
          tension: 0.4,
        },
      ],
    };
  });

  const pieChartData = derived(currentPlayer, ($player) => {
    if (!$player) return null;

    return {
      labels: ['现金', '库存商品', '升级投入'],
      datasets: [
        {
          data: [
            $player.gold || 0,
            calculateInventoryValue($player),
            Math.floor(($player.upgradeInvestment || 0) * 0.5),
          ],
          backgroundColor: ['#f59e0b', '#10b981', '#3b82f6'],
          borderWidth: 0,
        },
      ],
    };
  });

  const lineChartOptions = {
    responsive: true,
    plugins: {
      legend: {
        labels: { color: '#f8fafc' },
      },
      title: {
        display: true,
        text: '资产变化曲线',
        color: '#f8fafc',
        font: { size: 16 },
      },
    },
    scales: {
      x: {
        ticks: { color: '#94a3b8' },
        grid: { color: 'rgba(148, 163, 184, 0.1)' },
      },
      y: {
        ticks: { color: '#94a3b8' },
        grid: { color: 'rgba(148, 163, 184, 0.1)' },
      },
    },
  };

  const pieChartOptions = {
    responsive: true,
    plugins: {
      legend: {
        position: 'bottom',
        labels: { color: '#f8fafc' },
      },
      title: {
        display: true,
        text: '资产构成',
        color: '#f8fafc',
        font: { size: 16 },
      },
    },
  };

  function calculateInventoryValue(player) {
    let value = 0;
    for (const item of player.warehouse || []) {
      value += Math.floor(item.purchaseCost * 0.8);
    }
    for (const slot of player.shelves || []) {
      if (slot.item) {
        value += Math.floor(slot.item.purchaseCost * 0.8);
      }
    }
    return value;
  }

  onMount(() => {
    const stored = sessionStorage.getItem('gameResults');
    if (stored) {
      results = JSON.parse(stored);
    }

    const userId = $currentUser.playerId;
    if (results) {
      myResult = results.find(r => r.playerId === userId);
    }
  });

  function backToLobby() {
    navigate('/');
  }

  $: sortedResults = results ? [...results].sort((a, b) => b.finalAssets - a.finalAssets) : [];
</script>

<div class="settlement-container">
  <div class="settlement-header card">
    <h1>🏆 游戏结束</h1>
    {#if myResult}
      <div class="my-result">
        {#if myResult.isWinner}
          <div class="winner">🎉 恭喜获胜！</div>
        {:else}
          <div class="rank">排名: 第 {myResult.rank} 名</div>
        {/if}
        <div class="final-assets">最终资产: {myResult.finalAssets.toLocaleString()} 💰</div>
      </div>
    {/if}
  </div>

  <div class="settlement-content">
    <div class="rankings card">
      <h2>🏅 最终排名</h2>
      <div class="ranking-list">
        {#each sortedResults as result, idx}
          <div class="ranking-item {result.isWinner ? 'winner' : ''} {result.playerId === $currentUser.playerId ? 'me' : ''}">
            <span class="rank">#{idx + 1}</span>
            <span class="name">{result.name}</span>
            <span class="assets">{result.finalAssets.toLocaleString()} 💰</span>
            {#if result.isWinner}
              <span class="crown">👑</span>
            {/if}
          </div>
        {/each}
      </div>
    </div>

    <div class="charts">
      {#if $currentPlayer}
        <div class="chart-card card">
          {#if $lineChartData}
            <Line data={$lineChartData} options={lineChartOptions} />
          {/if}
        </div>
        <div class="chart-card card">
          {#if $pieChartData}
            <Pie data={$pieChartData} options={pieChartOptions} />
          {/if}
        </div>
      {/if}
    </div>

    {#if $currentPlayer}
      <div class="stats-summary card">
        <h2>📊 游戏统计</h2>
        <div class="stats-grid">
          <div class="stat-item">
            <span class="stat-label">最高资产</span>
            <span class="stat-value">{Math.max(...($currentPlayer.assetHistory || [0])).toLocaleString()} 💰</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">累计收入</span>
            <span class="stat-value">{($currentPlayer.weeklyStats?.income || 0).toLocaleString()} 💰</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">累计支出</span>
            <span class="stat-value">{($currentPlayer.weeklyStats?.expense || 0).toLocaleString()} 💰</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">雇佣冒险者</span>
            <span class="stat-value">{($currentPlayer.adventurers?.length || 0)} 人</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">货架数量</span>
            <span class="stat-value">{($currentPlayer.shelves?.length || 0)} 个</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">升级投入</span>
            <span class="stat-value">{($currentPlayer.upgradeInvestment || 0).toLocaleString()} 💰</span>
          </div>
        </div>
      </div>
    {/if}
  </div>

  <div class="actions">
    <button class="btn btn-primary large" on:click={backToLobby}>
      🏠 返回大厅
    </button>
  </div>
</div>

<style>
  .settlement-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 40px 20px;
  }

  .settlement-header {
    text-align: center;
    margin-bottom: 30px;
  }

  .settlement-header h1 {
    font-size: 48px;
    background: linear-gradient(135deg, var(--primary) 0%, var(--secondary) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    margin-bottom: 20px;
  }

  .my-result {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
  }

  .winner {
    font-size: 32px;
    color: var(--secondary);
    font-weight: bold;
    animation: bounce 1s ease infinite;
  }

  @keyframes bounce {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-10px); }
  }

  .rank {
    font-size: 24px;
    color: var(--primary);
    font-weight: 600;
  }

  .final-assets {
    font-size: 20px;
    color: var(--success);
    font-weight: 600;
  }

  .settlement-content {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
    margin-bottom: 30px;
  }

  .rankings h2 {
    color: var(--primary);
    margin-bottom: 20px;
  }

  .ranking-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .ranking-item {
    display: flex;
    align-items: center;
    gap: 15px;
    padding: 15px 20px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 8px;
    transition: all 0.3s ease;
  }

  .ranking-item.winner {
    background: linear-gradient(90deg, rgba(245, 158, 11, 0.2), transparent);
    border-left: 4px solid var(--secondary);
  }

  .ranking-item.me {
    border: 2px solid var(--primary);
  }

  .ranking-item .rank {
    font-size: 24px;
    font-weight: bold;
    color: var(--primary);
    min-width: 50px;
  }

  .ranking-item .name {
    flex: 1;
    font-weight: 600;
  }

  .ranking-item .assets {
    font-size: 18px;
    color: var(--secondary);
    font-weight: 600;
  }

  .ranking-item .crown {
    font-size: 24px;
    animation: rotate 3s linear infinite;
  }

  @keyframes rotate {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .charts {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .chart-card {
    padding: 20px;
  }

  .stats-summary h2 {
    color: var(--primary);
    margin-bottom: 20px;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 15px;
  }

  .stat-item {
    display: flex;
    flex-direction: column;
    gap: 5px;
    padding: 15px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 8px;
  }

  .stat-label {
    font-size: 12px;
    color: var(--gray);
  }

  .stat-value {
    font-size: 18px;
    font-weight: 600;
    color: var(--light);
  }

  .actions {
    text-align: center;
  }

  .btn.large {
    padding: 15px 40px;
    font-size: 18px;
  }

  @media (max-width: 768px) {
    .settlement-content {
      grid-template-columns: 1fr;
    }

    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
</style>
