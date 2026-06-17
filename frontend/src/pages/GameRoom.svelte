<script>
  import { onMount, onDestroy } from 'svelte';
  import { derived } from 'svelte/store';
  import { navigate } from 'svelte-routing';
  import {
    room,
    roomId,
    currentUser,
    currentPlayer,
    playerRanking,
    itemTypes,
    websocket,
    currentPhase,
    phaseEndTime,
    eventLog,
    gameResults,
    isConnected,
    addLog,
  } from '../stores/gameStore.js';
  import { api, connectWebSocket, sendWS } from '../utils/api.js';

  import PlayerBar from '../components/PlayerBar.svelte';
  import ShopPanel from '../components/ShopPanel.svelte';
  import ActionBar from '../components/ActionBar.svelte';
  import EventLog from '../components/EventLog.svelte';
  import WholesalerModal from '../components/WholesalerModal.svelte';
  import HireModal from '../components/HireModal.svelte';
  import UpgradeModal from '../components/UpgradeModal.svelte';
  import SynthesisModal from '../components/SynthesisModal.svelte';

  export let params;
  let ws = null;
  let countdown = 0;
  let countdownInterval = null;
  let showWholesaler = false;
  let showHire = false;
  let showUpgrade = false;
  let showSynthesis = false;
  let draggedItem = null;

  const phaseNames = {
    purchase: '进货日',
    business: '营业日',
    explore: '探索日',
    settlement: '结算',
  };

  const canAct = derived([currentPhase, currentPlayer], ([$phase, $player]) => {
    if (!$player || $player.isBankrupt) return false;
    return $phase === 'purchase' || $phase === 'explore';
  });

  onMount(async () => {
    roomId.set(params.roomId);
    
    try {
      await api.getItemTypes().then(types => itemTypes.set(types));
      const roomData = await api.getRoom(params.roomId);
      room.set(roomData);
    } catch (e) {
      console.error('Failed to load room:', e);
    }

    const user = $currentUser;
    if (!user.playerId) {
      navigate('/');
      return;
    }

    ws = connectWebSocket(params.roomId, user.playerId);
    websocket.set(ws);

    ws.onopen = () => {
      isConnected.set(true);
      addLog('已连接到游戏服务器', 'success');
    };

    ws.onclose = () => {
      isConnected.set(false);
      addLog('与服务器断开连接', 'error');
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        handleMessage(msg);
      } catch (e) {
        console.error('Failed to parse message:', e);
      }
    };

    countdownInterval = setInterval(updateCountdown, 1000);
  });

  onDestroy(() => {
    if (ws) ws.close();
    if (countdownInterval) clearInterval(countdownInterval);
  });

  function handleMessage(msg) {
    switch (msg.type) {
      case 'room_state':
      case 'room_update':
        room.set(msg.data);
        currentPhase.set(msg.data.phase);
        phaseEndTime.set(msg.data.phaseEndTime);
        break;
      case 'phase_update':
        currentPhase.set(msg.data.phase);
        phaseEndTime.set(msg.data.phaseEndTime);
        addLog(`进入 ${phaseNames[msg.data.phase]} 阶段 - 第 ${msg.data.week} 周`, 'info');
        break;
      case 'player_joined':
        addLog(`${msg.data.name} 加入了游戏`, 'info');
        break;
      case 'game_start':
        room.set(msg.data);
        currentPhase.set(msg.data.phase);
        phaseEndTime.set(msg.data.phaseEndTime);
        addLog('游戏开始！', 'success');
        break;
      case 'game_end':
        gameResults.set(msg.data);
        addLog('游戏结束！', 'success');
        setTimeout(() => navigate(`/room/${params.roomId}/settlement`), 2000);
        break;
      case 'chat':
        addLog(msg.data, 'chat');
        break;
    }
  }

  function updateCountdown() {
    const now = Math.floor(Date.now() / 1000);
    countdown = Math.max(0, $phaseEndTime - now);
  }

  async function startGame() {
    try {
      await api.startGame(params.roomId);
    } catch (e) {
      addLog(e.message, 'error');
    }
  }

  function buyItem(itemTypeId, quality, quantity) {
    sendWS(ws, 'buy_item', { itemTypeId, quality, quantity });
  }

  function placeItem(itemId, shelfId, price) {
    sendWS(ws, 'place_item', { itemId, shelfId, price });
  }

  function setPrice(shelfId, price) {
    sendWS(ws, 'set_price', { shelfId, price });
  }

  function removeItem(shelfId) {
    sendWS(ws, 'remove_item', { shelfId });
  }

  function hireAdventurer(adventurerIdx) {
    sendWS(ws, 'hire_adventurer', { adventurerIdx });
  }

  function dispatchAdventurer(adventurerId, floor) {
    sendWS(ws, 'dispatch_adventurer', { adventurerId, floor });
  }

  function startSynthesis(recipeId) {
    sendWS(ws, 'start_synthesis', { recipeId });
  }

  function upgradeShop(upgradeType) {
    sendWS(ws, 'upgrade_shop', { upgradeType });
  }

  function handleDragStart(item) {
    draggedItem = item;
  }

  function handleDrop(shelf) {
    if (draggedItem && $currentPhase === 'purchase') {
      const itemType = $itemTypes[draggedItem.typeId];
      const basePrice = itemType.basePrice;
      const qualityMult = { common: 1, fine: 1.5, rare: 2.5, legendary: 4 }[draggedItem.quality];
      const suggestedPrice = Math.floor(basePrice * qualityMult * 1.2);
      placeItem(draggedItem.id, shelf.id, suggestedPrice);
      addLog(`上架了 ${itemType.name}`, 'info');
    }
    draggedItem = null;
  }

  function leaveRoom() {
    if (confirm('确定要离开房间吗？')) {
      navigate('/');
    }
  }

  $: currentWeek = $room?.currentWeek || 0;
  $: totalWeeks = $room?.totalWeeks || 12;
  $: currentEvent = $room?.currentEvent;
  $: isHost = $room && Object.keys($room.players)[0] === $currentUser.playerId;
  $: canStart = $room?.status === 'waiting' && Object.keys($room.players).length >= 2 && isHost;
</script>

<div class="game-container">
  <div class="game-header">
    <div class="game-info">
      <h2>{$room?.name || '加载中...'}</h2>
      <div class="game-meta">
        {#if $room?.status === 'playing'}
          <span class="week">第 {$currentWeek}/{$totalWeeks} 周</span>
          <span class="phase phase-{$currentPhase}">{phaseNames[$currentPhase]}</span>
          <span class="countdown">⏱️ {countdown}s</span>
        {:else if $room?.status === 'waiting'}
          <span class="waiting">等待玩家加入...</span>
        {:else}
          <span class="finished">游戏已结束</span>
        {/if}
      </div>
      {#if currentEvent}
        <div class="event-banner">
          📢 {currentEvent.name}: {currentEvent.description}
        </div>
      {/if}
    </div>
    <button class="btn btn-danger small" on:click={leaveRoom}>离开</button>
  </div>

  {#if $room?.status === 'waiting'}
    <div class="waiting-room card">
      <h3>等待中的玩家</h3>
      <div class="waiting-players">
        {#each Object.values($room.players) as player}
          <div class="waiting-player">
            <span class="player-name">{player.name}</span>
            <span class="shop-name">{player.shopName}</span>
          </div>
        {/each}
      </div>
      <p class="player-count">{Object.keys($room.players).length}/{$room.maxPlayers} 人</p>
      {#if canStart}
        <button class="btn btn-success" on:click={startGame}>🎮 开始游戏</button>
      {:else if isHost}
        <p class="hint">至少需要 2 名玩家才能开始游戏</p>
      {:else}
        <p class="hint">等待房主开始游戏...</p>
      {/if}
    </div>
  {:else}
    <PlayerBar />

    <div class="game-main">
      <ShopPanel
        bind:showWholesaler
        bind:showHire
        bind:showSynthesis
        {handleDragStart}
        {handleDrop}
        {setPrice}
        {removeItem}
        {dispatchAdventurer}
      />
    </div>

    <ActionBar
      {showWholesaler}
      {showHire}
      {showUpgrade}
      {showSynthesis}
    />

    <EventLog />
  {/if}

  {#if showWholesaler}
    <WholesalerModal
      on:close={() => showWholesaler = false}
      {buyItem}
    />
  {/if}

  {#if showHire}
    <HireModal
      on:close={() => showHire = false}
      {hireAdventurer}
    />
  {/if}

  {#if showUpgrade}
    <UpgradeModal
      on:close={() => showUpgrade = false}
      {upgradeShop}
    />
  {/if}

  {#if showSynthesis}
    <SynthesisModal
      on:close={() => showSynthesis = false}
      {startSynthesis}
    />
  {/if}
</div>

<style>
  .game-container {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    padding: 10px;
    gap: 10px;
  }

  .game-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding: 15px;
    background: var(--card-bg);
    border-radius: 12px;
    border: 1px solid var(--border);
  }

  .game-info h2 {
    margin-bottom: 10px;
    color: var(--primary);
  }

  .game-meta {
    display: flex;
    gap: 15px;
    align-items: center;
  }

  .week {
    font-size: 16px;
    font-weight: 600;
  }

  .phase {
    padding: 4px 12px;
    border-radius: 20px;
    font-weight: 600;
    font-size: 14px;
  }

  .countdown {
    font-size: 18px;
    font-weight: bold;
    color: var(--secondary);
  }

  .waiting {
    color: var(--secondary);
  }

  .finished {
    color: var(--gray);
  }

  .event-banner {
    margin-top: 10px;
    padding: 10px 15px;
    background: linear-gradient(90deg, rgba(245, 158, 11, 0.2), transparent);
    border-left: 4px solid var(--secondary);
    border-radius: 4px;
    font-weight: 500;
  }

  .waiting-room {
    max-width: 500px;
    margin: 40px auto;
    text-align: center;
  }

  .waiting-room h3 {
    margin-bottom: 20px;
    color: var(--primary);
  }

  .waiting-players {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-bottom: 20px;
  }

  .waiting-player {
    display: flex;
    justify-content: space-between;
    padding: 12px;
    background: rgba(139, 92, 246, 0.1);
    border-radius: 8px;
  }

  .player-name {
    font-weight: 600;
  }

  .shop-name {
    color: var(--gray);
  }

  .player-count {
    margin-bottom: 20px;
    color: var(--gray);
  }

  .hint {
    color: var(--gray);
    margin-top: 10px;
  }

  .game-main {
    flex: 1;
    overflow-y: auto;
  }

  .btn.small {
    padding: 8px 16px;
    font-size: 12px;
  }
</style>
