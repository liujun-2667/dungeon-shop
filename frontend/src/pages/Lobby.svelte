<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { api } from '../utils/api.js';
  import { setUser } from '../stores/gameStore.js';

  let rooms = [];
  let showCreateModal = false;
  let showQuickMatch = false;
  let playerName = '';
  let shopName = '';
  let roomName = '';
  let maxPlayers = 4;
  let loading = false;
  let error = '';

  const shopNames = ['神秘洞穴', '冒险者之家', '地牢补给站', '魔法杂货铺', '勇士驿站', '珍宝阁'];

  onMount(async () => {
    await loadRooms();
    setInterval(loadRooms, 5000);
  });

  async function loadRooms() {
    try {
      rooms = await api.listRooms();
    } catch (e) {
      console.error('Failed to load rooms:', e);
    }
  }

  async function createRoom() {
    if (!playerName.trim() || !roomName.trim()) {
      error = '请填写所有必填项';
      return;
    }

    if (!shopName.trim()) {
      shopName = shopNames[Math.floor(Math.random() * shopNames.length)];
    }

    loading = true;
    error = '';

    try {
      const result = await api.createRoom(roomName, maxPlayers, playerName, shopName);
      setUser(result.playerId, playerName, shopName);
      navigate(`/room/${result.roomId}`);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function quickMatch() {
    if (!playerName.trim()) {
      error = '请输入玩家名称';
      return;
    }

    if (!shopName.trim()) {
      shopName = shopNames[Math.floor(Math.random() * shopNames.length)];
    }

    loading = true;
    error = '';

    try {
      const availableRooms = rooms.filter(r => r.status === 'waiting' && Object.keys(r.players).length < r.maxPlayers);
      
      if (availableRooms.length > 0) {
        const room = availableRooms[Math.floor(Math.random() * availableRooms.length)];
        const result = await api.joinRoom(room.id, playerName, shopName);
        setUser(result.playerId, playerName, shopName);
        navigate(`/room/${result.roomId}`);
      } else {
        const result = await api.createRoom('快速匹配房间', 4, playerName, shopName);
        setUser(result.playerId, playerName, shopName);
        navigate(`/room/${result.roomId}`);
      }
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function joinRoom(roomId) {
    if (!playerName.trim()) {
      error = '请先输入玩家名称';
      return;
    }

    if (!shopName.trim()) {
      shopName = shopNames[Math.floor(Math.random() * shopNames.length)];
    }

    loading = true;
    error = '';

    try {
      const result = await api.joinRoom(roomId, playerName, shopName);
      setUser(result.playerId, playerName, shopName);
      navigate(`/room/${result.roomId}`);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function getPlayerCount(room) {
    return Object.keys(room.players).length;
  }
</script>

<div class="lobby-container">
  <header class="lobby-header">
    <h1>🏰 地牢商店</h1>
    <p class="subtitle">Dungeon Shop Tycoon</p>
  </header>

  <div class="lobby-content">
    <div class="user-setup card">
      <h2>玩家信息</h2>
      <div class="form-group">
        <label>玩家名称</label>
        <input
          type="text"
          bind:value={playerName}
          placeholder="输入你的名字"
          maxlength={12}
        />
      </div>
      <div class="form-group">
        <label>商店名称 (可选)</label>
        <input
          type="text"
          bind:value={shopName}
          placeholder="随机生成"
          maxlength={12}
        />
      </div>
      {#if error}
        <p class="error">{error}</p>
      {/if}
    </div>

    <div class="action-buttons">
      <button class="btn btn-primary" on:click={() => showCreateModal = true}>
        ➕ 创建房间
      </button>
      <button class="btn btn-secondary" on:click={quickMatch} disabled={loading || !playerName.trim()}>
        ⚡ 快速匹配
      </button>
    </div>

    <div class="room-list card">
      <div class="room-list-header">
        <h2>房间列表</h2>
        <button class="btn btn-primary small" on:click={loadRooms}>🔄 刷新</button>
      </div>
      
      {#if rooms.length === 0}
        <p class="empty">暂无可用房间，创建一个吧！</p>
      {:else}
        <div class="rooms">
          {#each rooms as room}
            <div class="room-card">
              <div class="room-info">
                <h3>{room.name}</h3>
                <p class="room-meta">
                  <span class="status {room.status}">
                    {room.status === 'waiting' ? '等待中' : room.status === 'playing' ? '游戏中' : '已结束'}
                  </span>
                  <span class="players">{getPlayerCount(room)}/{room.maxPlayers} 人</span>
                </p>
              </div>
              <button
                class="btn btn-success small"
                on:click={() => joinRoom(room.id)}
                disabled={room.status !== 'waiting' || getPlayerCount(room) >= room.maxPlayers || loading}
              >
                {room.status === 'waiting' ? '加入' : '观战'}
              </button>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  {#if showCreateModal}
    <div class="modal-overlay" on:click={() => showCreateModal = false}>
      <div class="modal card" on:click|stopPropagation>
        <h2>创建房间</h2>
        <div class="form-group">
          <label>房间名称</label>
          <input type="text" bind:value={roomName} placeholder="我的地牢商店" maxlength={20} />
        </div>
        <div class="form-group">
          <label>最大玩家数</label>
          <select bind:value={maxPlayers}>
            <option value={2}>2 人</option>
            <option value={3}>3 人</option>
            <option value={4}>4 人</option>
          </select>
        </div>
        <div class="modal-actions">
          <button class="btn btn-danger" on:click={() => showCreateModal = false}>取消</button>
          <button class="btn btn-primary" on:click={createRoom} disabled={loading}>
            {loading ? '创建中...' : '创建房间'}
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .lobby-container {
    max-width: 900px;
    margin: 0 auto;
    padding: 40px 20px;
  }

  .lobby-header {
    text-align: center;
    margin-bottom: 40px;
  }

  .lobby-header h1 {
    font-size: 48px;
    background: linear-gradient(135deg, var(--primary) 0%, var(--secondary) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    margin-bottom: 10px;
  }

  .subtitle {
    color: var(--gray);
    font-size: 18px;
  }

  .lobby-content {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .user-setup h2 {
    margin-bottom: 20px;
    color: var(--primary);
  }

  .form-group {
    margin-bottom: 15px;
  }

  .form-group label {
    display: block;
    margin-bottom: 8px;
    font-weight: 600;
    color: var(--light);
  }

  .form-group input,
  .form-group select {
    width: 100%;
  }

  .error {
    color: var(--danger);
    margin-top: 10px;
    font-size: 14px;
  }

  .action-buttons {
    display: flex;
    gap: 15px;
  }

  .action-buttons .btn {
    flex: 1;
    padding: 15px;
    font-size: 16px;
  }

  .room-list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }

  .room-list-header h2 {
    color: var(--primary);
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

  .rooms {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .room-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px;
    background: rgba(139, 92, 246, 0.1);
    border-radius: 8px;
    border: 1px solid var(--border);
  }

  .room-info h3 {
    margin-bottom: 5px;
  }

  .room-meta {
    display: flex;
    gap: 15px;
    font-size: 14px;
    color: var(--gray);
  }

  .status {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
  }

  .status.waiting {
    background: rgba(16, 185, 129, 0.2);
    color: var(--success);
  }

  .status.playing {
    background: rgba(245, 158, 11, 0.2);
    color: var(--secondary);
  }

  .status.finished {
    background: rgba(100, 116, 139, 0.2);
    color: var(--gray);
  }

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
    max-width: 400px;
  }

  .modal h2 {
    margin-bottom: 20px;
    color: var(--primary);
  }

  .modal-actions {
    display: flex;
    gap: 10px;
    margin-top: 20px;
  }

  .modal-actions .btn {
    flex: 1;
  }
</style>
