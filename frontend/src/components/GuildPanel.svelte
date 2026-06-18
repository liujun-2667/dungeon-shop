<script>
  import { createEventDispatcher } from 'svelte';
  import {
    room,
    currentUser,
    currentPlayer,
    itemTypes,
    guilds,
    getGuildWarehouseCapacity,
    getGuildUpgradeCost,
    getGuildMaxMembers,
    isGuildLeader,
  } from '../stores/gameStore.js';
  import { sendWS } from '../utils/api.js';

  const dispatch = createEventDispatcher();

  export let ws;

  let activeTab = 'info';
  let showCreateForm = false;
  let newGuildName = '';
  let draggedItem = null;
  let selectedItem = null;

  const qualityNames = {
    common: '普通',
    fine: '精良',
    rare: '稀有',
    legendary: '传说',
  };

  $: playerId = $currentUser.playerId;
  $: playerGuildId = $currentPlayer?.guildId || '';
  $: currentGuild = $guilds.find(g => g.id === playerGuildId) || null;
  $: warehouseCapacity = currentGuild ? getGuildWarehouseCapacity(currentGuild.level) : 0;
  $: upgradeCost = currentGuild ? getGuildUpgradeCost(currentGuild.level) : 0;
  $: canUpgrade = currentGuild && isGuildLeader(currentGuild, playerId) && currentGuild.level < 5 && currentGuild.treasury >= upgradeCost;
  $: maxMembers = $room ? getGuildMaxMembers($room.maxPlayers) : 0;
  $: isLeader = currentGuild && isGuildLeader(currentGuild, playerId);

  function getItemInfo(typeId) {
    return $itemTypes[typeId] || { name: typeId, category: '' };
  }

  function createGuild() {
    if (!newGuildName || newGuildName.trim().length < 2) {
      alert('公会名称至少需要2个字符');
      return;
    }
    if ($currentPlayer.gold < 50) {
      alert('金币不足，创建公会需要50金币');
      return;
    }
    sendWS(ws, 'create_guild', { guildName: newGuildName.trim() });
    showCreateForm = false;
    newGuildName = '';
  }

  function joinGuild(guildId) {
    if (!confirm('确定要加入这个公会吗？')) return;
    sendWS(ws, 'join_guild', { guildId });
  }

  function leaveGuild() {
    if (!confirm('确定要退出公会吗？')) return;
    sendWS(ws, 'leave_guild', {});
  }

  function kickMember(targetPlayerId) {
    if (!confirm('确定要踢出这名成员吗？踢出后2周内无法重新加入。')) return;
    sendWS(ws, 'kick_guild_member', { targetPlayerId });
  }

  function upgradeGuild() {
    if (!confirm(`确定花费 ${upgradeCost}💰 升级公会吗？`)) return;
    sendWS(ws, 'upgrade_guild', {});
  }

  function depositItem(itemId) {
    sendWS(ws, 'deposit_guild_warehouse', { itemId });
  }

  function withdrawItem(itemId) {
    sendWS(ws, 'withdraw_guild_warehouse', { itemId });
  }

  function onDragStart(item, source) {
    draggedItem = { item, source };
    selectedItem = { item, source };
  }

  function onDragOver(e) {
    e.preventDefault();
  }

  function onDrop(target) {
    if (!draggedItem) return;
    
    if (target === 'guild' && draggedItem.source === 'personal') {
      depositItem(draggedItem.item.id);
    } else if (target === 'personal' && draggedItem.source === 'guild') {
      withdrawItem(draggedItem.item.id);
    }
    
    draggedItem = null;
    selectedItem = null;
  }

  function selectItem(item, source) {
    if (selectedItem && selectedItem.item.id === item.id && selectedItem.source === source) {
      selectedItem = null;
    } else {
      selectedItem = { item, source };
    }
  }

  function transferToGuild() {
    if (!selectedItem) {
      alert('请先点击选择一件个人仓库中的物品');
      return;
    }
    if (selectedItem.source !== 'personal') {
      alert('请选择个人仓库中的物品进行存入');
      return;
    }
    depositItem(selectedItem.item.id);
    selectedItem = null;
  }

  function transferToPersonal() {
    if (!selectedItem) {
      alert('请先点击选择一件公会仓库中的物品');
      return;
    }
    if (selectedItem.source !== 'guild') {
      alert('请选择公会仓库中的物品进行取出');
      return;
    }
    withdrawItem(selectedItem.item.id);
    selectedItem = null;
  }

  function close() {
    dispatch('close');
  }
</script>

<div class="modal-overlay" on:click={close}>
  <div class="modal card guild-panel" on:click|stopPropagation>
    <div class="modal-header">
      <h2>🏰 公会系统</h2>
      {#if currentGuild}
        <div class="guild-header-info">
          <span class="guild-tag">[{currentGuild.abbreviation}]</span>
          <span class="guild-name">{currentGuild.name}</span>
          <span class="guild-level">Lv.{currentGuild.level}</span>
        </div>
      {/if}
      <button class="close-btn" on:click={close}>×</button>
    </div>

    {#if !currentGuild}
      <div class="no-guild-section">
        <div class="section-title">
          <h3>创建公会</h3>
          {#if !showCreateForm}
            <button class="btn btn-primary" on:click={() => showCreateForm = true}>
              创建公会 (50💰)
            </button>
          {/if}
        </div>

        {#if showCreateForm}
          <div class="create-form card">
            <input 
              type="text" 
              placeholder="输入公会名称" 
              bind:value={newGuildName}
              maxlength={12}
            />
            <div class="form-actions">
              <button class="btn btn-primary" on:click={createGuild} disabled={!newGuildName}>
                确认创建
              </button>
              <button class="btn btn-secondary" on:click={() => showCreateForm = false}>
                取消
              </button>
            </div>
          </div>
        {/if}

        <div class="section-title">
          <h3>已有公会</h3>
          <span class="hint">点击加入</span>
        </div>

        {#if $guilds.length === 0}
          <p class="empty">暂无公会，创建第一个公会吧！</p>
        {:else}
          <div class="guild-list">
            {#each $guilds as guild}
              <div class="guild-card">
                <div class="guild-info">
                  <span class="guild-tag">[{guild.abbreviation}]</span>
                  <span class="guild-name">{guild.name}</span>
                  <span class="guild-level">Lv.{guild.level}</span>
                  <span class="member-count">
                    {guild.members.length}/{maxMembers}人
                  </span>
                </div>
                <div class="guild-members">
                  {#each guild.members as member}
                    <span class="member-name" class:leader={member.isLeader}>
                      {member.isLeader ? '👑' : ''}{member.playerName}
                    </span>
                  {/each}
                </div>
                <button 
                  class="btn btn-primary small" 
                  on:click={() => joinGuild(guild.id)}
                  disabled={guild.members.length >= maxMembers}
                >
                  {guild.members.length >= maxMembers ? '已满' : '加入'}
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {:else}
      <div class="tab-bar">
        <button class="tab-btn" class:active={activeTab === 'info'} on:click={() => activeTab = 'info'}>
          公会信息
        </button>
        <button class="tab-btn" class:active={activeTab === 'warehouse'} on:click={() => activeTab = 'warehouse'}>
          共享仓库
        </button>
        <button class="tab-btn" class:active={activeTab === 'members'} on:click={() => activeTab = 'members'}>
          成员列表
        </button>
        {#if isLeader && currentGuild.level < 5}
          <button class="tab-btn upgrade-btn" class:disabled={!canUpgrade} on:click={upgradeGuild}>
            🔼 升级公会 ({upgradeCost}💰)
          </button>
        {/if}
        <button class="tab-btn leave-btn" on:click={leaveGuild}>
          🚪 退出公会
        </button>
      </div>

      <div class="modal-content">
        {#if activeTab === 'info'}
          <div class="guild-info-section">
            <div class="info-grid">
              <div class="info-item">
                <span class="label">公会名称</span>
                <span class="value">{currentGuild.name}</span>
              </div>
              <div class="info-item">
                <span class="label">公会缩写</span>
                <span class="value">[{currentGuild.abbreviation}]</span>
              </div>
              <div class="info-item">
                <span class="label">公会等级</span>
                <span class="value">Lv.{currentGuild.level}</span>
              </div>
              <div class="info-item">
                <span class="label">成员数量</span>
                <span class="value">{currentGuild.members.length}/{maxMembers}</span>
              </div>
              <div class="info-item">
                <span class="label">公会金库</span>
                <span class="value treasury">{currentGuild.treasury}💰</span>
              </div>
              <div class="info-item">
                <span class="label">仓库容量</span>
                <span class="value">{currentGuild.warehouse.length}/{warehouseCapacity}</span>
              </div>
            </div>

            {#if currentGuild.level < 5}
              <div class="upgrade-info">
                <h4>升级到 Lv.{currentGuild.level + 1}</h4>
                <p>需要金库: {upgradeCost}💰 (当前: {currentGuild.treasury}💰)</p>
                <p>升级后仓库容量: {getGuildWarehouseCapacity(currentGuild.level + 1)}格</p>
                <div class="progress-bar">
                  <div 
                    class="progress-fill" 
                    style="width: {Math.min(100, (currentGuild.treasury / upgradeCost) * 100)}%"
                  ></div>
                </div>
              </div>
            {/if}

            <div class="contribution-info">
              <h4>贡献说明</h4>
              <p>成员在拍卖行成功卖出商品时，成交价的2%自动作为公会贡献金充入公会金库。</p>
            </div>
          </div>

        {:else if activeTab === 'warehouse'}
          <div class="warehouse-section">
            <div class="warehouse-container">
              <div 
                class="warehouse-block"
                on:dragover={onDragOver}
                on:drop={() => onDrop('personal')}
              >
                <h4>个人仓库 (点击选中或拖拽取出)</h4>
                <div class="warehouse-grid">
                  {#each $currentPlayer.warehouse as item}
                    <div 
                      class="warehouse-item quality-{item.quality}"
                      class:selected={selectedItem && selectedItem.item.id === item.id && selectedItem.source === 'personal'}
                      draggable="true"
                      on:dragstart={() => onDragStart(item, 'personal')}
                      on:click={() => selectItem(item, 'personal')}
                      title="{getItemInfo(item.typeId).name} ({qualityNames[item.quality]})"
                    >
                      <span class="item-icon">📦</span>
                      <span class="item-name">{getItemInfo(item.typeId).name}</span>
                    </div>
                  {/each}
                  {#if $currentPlayer.warehouse.length === 0}
                    <p class="empty">个人仓库为空</p>
                  {/if}
                </div>
              </div>

              <div class="transfer-arrows">
                <button class="arrow" on:click={transferToGuild} title="将选中的个人物品存入公会仓库">
                  ➡️
                </button>
                <button class="arrow" on:click={transferToPersonal} title="将选中的公会物品取出到个人仓库">
                  ⬅️
                </button>
              </div>

              <div 
                class="warehouse-block guild-warehouse"
                on:dragover={onDragOver}
                on:drop={() => onDrop('guild')}
              >
                <h4>公会仓库 (点击选中或拖拽存入)</h4>
                <div class="warehouse-grid">
                  {#each currentGuild.warehouse as item}
                    <div 
                      class="warehouse-item quality-{item.quality}"
                      class:selected={selectedItem && selectedItem.item.id === item.id && selectedItem.source === 'guild'}
                      draggable="true"
                      on:dragstart={() => onDragStart(item, 'guild')}
                      on:click={() => selectItem(item, 'guild')}
                      title="{getItemInfo(item.typeId).name} ({qualityNames[item.quality]})"
                    >
                      <span class="item-icon">📦</span>
                      <span class="item-name">{getItemInfo(item.typeId).name}</span>
                    </div>
                  {/each}
                  {#each Array(warehouseCapacity - currentGuild.warehouse.length) as _, i}
                    <div class="warehouse-slot" key={i}>
                      <span class="slot-empty">空</span>
                    </div>
                  {/each}
                </div>
              </div>
            </div>
            <p class="warehouse-hint">💡 提示：点击物品选中后按箭头转移，或直接拖拽物品在个人仓库和公会仓库之间转移</p>
          </div>

        {:else if activeTab === 'members'}
          <div class="members-section">
            <div class="members-list">
              {#each currentGuild.members as member, i}
                <div class="member-card" class:leader={member.isLeader}>
                  <div class="member-info">
                    <span class="member-rank">#{i + 1}</span>
                    <span class="member-name">
                      {member.isLeader ? '👑' : ''}{member.playerName}
                    </span>
                    {#if member.isLeader}
                      <span class="leader-badge">会长</span>
                    {/if}
                  </div>
                  <div class="member-stats">
                    <span class="contribution">贡献: {member.contribution}💰</span>
                    <span class="join-time">
                      加入: {new Date(member.joinTime * 1000).toLocaleDateString()}
                    </span>
                  </div>
                  {#if isLeader && !member.isLeader}
                    <button 
                      class="btn btn-danger small" 
                      on:click={() => kickMember(member.playerId)}
                    >
                      踢出
                    </button>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .guild-panel {
    width: 95%;
    max-width: 900px;
    max-height: 85vh;
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 15px;
    padding-bottom: 12px;
    border-bottom: 1px solid var(--border);
    flex-wrap: wrap;
    gap: 10px;
  }

  .modal-header h2 {
    color: var(--primary);
    margin: 0;
  }

  .guild-header-info {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .guild-tag {
    background: linear-gradient(135deg, var(--primary), #6d28d9);
    color: white;
    padding: 4px 10px;
    border-radius: 6px;
    font-weight: bold;
    font-size: 14px;
  }

  .guild-name {
    font-weight: 600;
    font-size: 16px;
  }

  .guild-level {
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

  .no-guild-section {
    flex: 1;
    overflow-y: auto;
  }

  .section-title {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin: 20px 0 15px;
  }

  .section-title h3 {
    color: var(--primary);
    margin: 0;
  }

  .hint {
    color: var(--gray);
    font-size: 13px;
  }

  .create-form {
    padding: 20px;
    background: rgba(139, 92, 246, 0.1);
    border: 1px solid var(--primary);
    margin-bottom: 20px;
  }

  .create-form input {
    width: 100%;
    padding: 10px;
    margin-bottom: 10px;
    font-size: 14px;
  }

  .form-actions {
    display: flex;
    gap: 10px;
    justify-content: flex-end;
  }

  .guild-list {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 15px;
  }

  .guild-card {
    background: rgba(0, 0, 0, 0.2);
    padding: 15px;
    border-radius: 10px;
    border: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .guild-info {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .guild-info .guild-tag {
    padding: 2px 8px;
    font-size: 12px;
  }

  .guild-info .guild-name {
    font-weight: 600;
  }

  .guild-info .guild-level {
    color: var(--secondary);
    font-size: 13px;
  }

  .member-count {
    margin-left: auto;
    color: var(--gray);
    font-size: 13px;
  }

  .guild-members {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .member-name {
    background: rgba(139, 92, 246, 0.2);
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
  }

  .member-name.leader {
    background: rgba(245, 158, 11, 0.2);
    color: var(--secondary);
  }

  .tab-bar {
    display: flex;
    gap: 8px;
    margin-bottom: 15px;
    flex-wrap: wrap;
  }

  .tab-btn {
    padding: 8px 16px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: rgba(0, 0, 0, 0.3);
    color: var(--light);
    cursor: pointer;
    font-size: 13px;
    font-weight: 600;
    transition: all 0.2s;
  }

  .tab-btn.active {
    background: var(--primary);
    border-color: var(--primary);
    color: white;
  }

  .tab-btn:hover:not(.active):not(.disabled) {
    background: rgba(139, 92, 246, 0.2);
  }

  .tab-btn.disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .upgrade-btn {
    margin-left: auto;
    background: linear-gradient(135deg, var(--secondary), #d97706);
    border-color: var(--secondary);
    color: white;
  }

  .leave-btn {
    background: rgba(239, 68, 68, 0.2);
    border-color: var(--danger);
    color: var(--danger);
  }

  .leave-btn:hover {
    background: rgba(239, 68, 68, 0.3) !important;
  }

  .modal-content {
    flex: 1;
    overflow-y: auto;
  }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 15px;
    margin-bottom: 20px;
  }

  .info-item {
    background: rgba(0, 0, 0, 0.2);
    padding: 15px;
    border-radius: 8px;
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .info-item .label {
    color: var(--gray);
    font-size: 13px;
  }

  .info-item .value {
    font-size: 18px;
    font-weight: 600;
  }

  .info-item .value.treasury {
    color: var(--secondary);
  }

  .upgrade-info, .contribution-info {
    background: rgba(139, 92, 246, 0.1);
    padding: 15px;
    border-radius: 10px;
    margin-bottom: 15px;
  }

  .upgrade-info h4, .contribution-info h4 {
    color: var(--primary);
    margin: 0 0 10px;
  }

  .upgrade-info p {
    margin: 5px 0;
  }

  .progress-bar {
    height: 10px;
    background: rgba(0, 0, 0, 0.3);
    border-radius: 5px;
    overflow: hidden;
    margin-top: 10px;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--secondary), #f59e0b);
    transition: width 0.3s;
  }

  .warehouse-section {
    display: flex;
    flex-direction: column;
    gap: 15px;
  }

  .warehouse-container {
    display: flex;
    gap: 20px;
    align-items: stretch;
  }

  .warehouse-block {
    flex: 1;
    background: rgba(0, 0, 0, 0.2);
    padding: 15px;
    border-radius: 10px;
    border: 2px dashed transparent;
    transition: all 0.2s;
  }

  .warehouse-block:hover {
    border-color: var(--primary);
  }

  .warehouse-block.guild-warehouse {
    background: rgba(139, 92, 246, 0.1);
  }

  .warehouse-block h4 {
    color: var(--primary);
    margin: 0 0 10px;
  }

  .warehouse-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
    gap: 8px;
    min-height: 200px;
  }

  .warehouse-item {
    background: rgba(0, 0, 0, 0.3);
    padding: 8px;
    border-radius: 6px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    cursor: grab;
    transition: all 0.2s;
    border-left: 3px solid var(--primary);
  }

  .warehouse-item:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }

  .warehouse-item.quality-fine { border-left-color: #22c55e; }
  .warehouse-item.quality-rare { border-left-color: #3b82f6; }
  .warehouse-item.quality-legendary { border-left-color: #f59e0b; }

  .warehouse-item:active {
    cursor: grabbing;
  }

  .item-icon {
    font-size: 24px;
  }

  .item-name {
    font-size: 10px;
    text-align: center;
    line-height: 1.2;
  }

  .warehouse-slot {
    background: rgba(0, 0, 0, 0.1);
    padding: 8px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 60px;
  }

  .slot-empty {
    color: var(--gray);
    font-size: 12px;
  }

  .warehouse-item.selected {
    border: 2px solid var(--primary);
    box-shadow: 0 0 12px rgba(139, 92, 246, 0.6);
    background: rgba(139, 92, 246, 0.25);
    transform: translateY(-2px);
  }

  .transfer-arrows {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 20px;
  }

  .arrow {
    font-size: 24px;
    cursor: pointer;
    padding: 12px 16px;
    border: none;
    border-radius: 50%;
    background: rgba(139, 92, 246, 0.2);
    transition: all 0.2s;
    color: var(--light);
    line-height: 1;
  }

  .arrow:hover {
    background: var(--primary);
    transform: scale(1.1);
  }

  .arrow:active {
    transform: scale(0.95);
  }

  .warehouse-hint {
    text-align: center;
    color: var(--gray);
    font-size: 13px;
    margin: 0;
  }

  .members-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .member-card {
    background: rgba(0, 0, 0, 0.2);
    padding: 15px;
    border-radius: 10px;
    display: flex;
    align-items: center;
    gap: 15px;
    border-left: 3px solid var(--border);
  }

  .member-card.leader {
    border-left-color: var(--secondary);
    background: rgba(245, 158, 11, 0.1);
  }

  .member-info {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .member-rank {
    color: var(--gray);
    font-weight: 600;
    min-width: 30px;
  }

  .member-info .member-name {
    font-weight: 600;
    font-size: 15px;
  }

  .leader-badge {
    background: linear-gradient(135deg, var(--secondary), #d97706);
    color: white;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
  }

  .member-stats {
    display: flex;
    gap: 20px;
    margin-right: 15px;
  }

  .contribution {
    color: var(--secondary);
    font-weight: 600;
  }

  .join-time {
    color: var(--gray);
    font-size: 12px;
  }

  .empty {
    text-align: center;
    color: var(--gray);
    padding: 40px;
  }

  .btn.small {
    padding: 6px 12px;
    font-size: 12px;
  }

  .btn-danger {
    background: linear-gradient(135deg, #ef4444, #dc2626);
    color: white;
    border: none;
  }

  .btn-danger:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
  }
</style>
