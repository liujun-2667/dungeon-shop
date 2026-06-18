<script>
  import { onMount, onDestroy } from 'svelte';
  import { derived, get } from 'svelte/store';
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
    addBargainRequest,
    removeBargainRequest,
    clearBargainRequests,
    addAuctionError,
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
  import BargainBubble from '../components/BargainBubble.svelte';
  import AuctionHouse from '../components/AuctionHouse.svelte';

  export let params;
  let ws = null;
  let countdown = 0;
  let countdownInterval = null;
  let showWholesaler = false;
  let showHire = false;
  let showUpgrade = false;
  let showSynthesis = false;
  let showAuctionHouse = false;
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

    const user = get(currentUser);
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
    clearBargainRequests();
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
        if (msg.data.phase !== 'business') {
          clearBargainRequests();
        }
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
        clearBargainRequests();
        setTimeout(() => navigate(`/room/${params.roomId}/settlement`), 2000);
        break;
      case 'chat':
        addLog(msg.data, 'chat');
        break;
      case 'business_log':
        if (msg.data && msg.data.message) {
          const logType = msg.data.type === 'purchase' || msg.data.type === 'bargain_success' || msg.data.type === 'bargain_reject_buy'
            ? 'success'
            : msg.data.type === 'bargain_start'
              ? 'info'
              : 'info';
          addLog(msg.data.message, logType);
        }
        break;
      case 'bargain_request':
        if (msg.data) {
          addBargainRequest(msg.data);
          addLog(`⚠️ ${msg.data.npcName} 发起了砍价！`, 'warning');
        }
        break;
      case 'bargain_timeout':
        if (msg.data) {
          removeBargainRequest(msg.data);
          addLog('砍价超时，已默认拒绝', 'warning');
        }
        break;
      case 'bargain_resolved':
        if (msg.data && msg.data.bargainId) {
          removeBargainRequest(msg.data.bargainId);
        }
        break;
      case 'bid_update':
        if (msg.data) {
          const depositText = msg.data.deposit ? `，保证金 ${msg.data.deposit}💰` : '';
          addLog(`🏷️ 拍卖出价更新: ${msg.data.itemTypeName || '商品'} 当前价 ${msg.data.currentPrice}💰${depositText} (${msg.data.highestBidder})`, 'info');
        }
        break;
      case 'auction_end':
        if (msg.data) {
          const statusText = msg.data.status === 'sold' ? '已成交' : '已流拍';
          addLog(`🏛️ 拍卖结束: ${msg.data.itemTypeName || '商品'} ${statusText}，成交价 ${msg.data.currentPrice}💰`, msg.data.status === 'sold' ? 'success' : 'warning');
        }
        break;
      case 'buyout':
        if (msg.data) {
          addLog(`🏛️ 一口价成交: ${msg.data.itemTypeName || '商品'} 以 ${msg.data.currentPrice}💰 被买走`, 'success');
        }
        break;
      case 'auction_created':
        if (msg.data) {
          addLog(`📦 新拍卖上架: ${msg.data.itemTypeName || '商品'}`, 'info');
        }
        break;
      case 'auction_cancelled':
        addLog('拍卖已取消', 'info');
        break;
      case 'auction_error':
        if (msg.data) {
          addLog(`⚠️ 拍卖操作失败: ${msg.data.error}`, 'error');
          addAuctionError(msg.data.action, msg.data.error);
        }
        break;
      case 'reputation_update':
        if (msg.data && Array.isArray(msg.data)) {
          room.update(currentRoom => {
            if (!currentRoom || !currentRoom.players) return currentRoom;
            const updatedRoom = { ...currentRoom };
            for (const rep of msg.data) {
              if (updatedRoom.players[rep.playerId]) {
                updatedRoom.players[rep.playerId] = {
                  ...updatedRoom.players[rep.playerId],
                  auctionReputation: rep.auctionReputation,
                  reputation: rep.shopReputation,
                };
              }
            }
            return updatedRoom;
          });
        }
        break;
    }
  }

  function updateCountdown() {
    const now = Math.floor(Date.now() / 1000);
    countdown = Math.max(0, get(phaseEndTime) - now);
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
    if (draggedItem && get(currentPhase) === 'purchase') {
      const itemType = get(itemTypes)[draggedItem.typeId];
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
          <span class="week">第 {currentWeek}/{totalWeeks} 周</span>
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
      bind:showWholesaler
      bind:showHire
      bind:showUpgrade
      bind:showSynthesis
      bind:showAuctionHouse
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

  {#if showAuctionHouse}
    <AuctionHouse {ws} on:close={() => showAuctionHouse = false} />
  {/if}

  <BargainBubble {ws} />
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
