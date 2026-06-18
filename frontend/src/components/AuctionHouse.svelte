<script>
  import { createEventDispatcher } from 'svelte';
  import {
    room,
    currentUser,
    currentPlayer,
    currentPhase,
    itemTypes,
    addLog,
    getActiveAuctions,
    getMyAuctions,
    getMyBids,
    getMinBid,
    getRemainingWeeks,
    getAuctionStatusText,
  } from '../stores/gameStore.js';
  import { sendWS } from '../utils/api.js';

  const dispatch = createEventDispatcher();

  export let ws;

  let activeTab = 'browse';
  let filterCategory = '';
  let filterQuality = '';
  let sortBy = 'price_asc';
  let bidAmount = '';
  let bidAuctionId = '';
  let showBidConfirm = false;
  let selectedAuction = null;
  let showCreateForm = false;
  let createItemId = '';
  let createStartingPrice = '';
  let createBuyoutPrice = '';

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

  const sortOptions = [
    { value: 'price_asc', label: '价格↑' },
    { value: 'price_desc', label: '价格↓' },
    { value: 'time_asc', label: '剩余时间↑' },
    { value: 'time_desc', label: '剩余时间↓' },
    { value: 'quality_desc', label: '品质↓' },
  ];

  $: playerId = $currentUser.playerId;
  $: currentWeek = $room?.currentWeek || 0;
  $: activeAuctions = getActiveAuctions($room);
  $: myAuctions = getMyAuctions($room, playerId);
  $: myBids = getMyBids($room, playerId);
  $: availableGold = $currentPlayer?.gold || 0;
  $: frozenGold = $currentPlayer?.frozenGold || 0;

  $: filteredAuctions = activeAuctions
    .filter(a => {
      if (filterCategory) {
        const itemType = $itemTypes[a.item.typeId];
        if (!itemType || itemType.category !== filterCategory) return false;
      }
      if (filterQuality && a.item.quality !== filterQuality) return false;
      return true;
    })
    .sort((a, b) => {
      switch (sortBy) {
        case 'price_asc': return a.currentPrice - b.currentPrice;
        case 'price_desc': return b.currentPrice - a.currentPrice;
        case 'time_asc': return (a.endWeek - currentWeek) - (b.endWeek - currentWeek);
        case 'time_desc': return (b.endWeek - currentWeek) - (a.endWeek - currentWeek);
        case 'quality_desc': {
          const qr = { common: 0, fine: 1, rare: 2, legendary: 3 };
          return (qr[b.item.quality] || 0) - (qr[a.item.quality] || 0);
        }
        default: return 0;
      }
    });

  $: canListAuction = $currentPhase === 'purchase' && !$currentPlayer?.isBankrupt;
  $: canBid = ($currentPhase === 'business' || $currentPhase === 'explore') && !$currentPlayer?.isBankrupt;

  $: warehouseItems = $currentPlayer?.warehouse || [];

  function getItemInfo(typeId) {
    return $itemTypes[typeId] || { name: typeId, category: '' };
  }

  function openBidConfirm(auction) {
    const min = getMinBid(auction);
    bidAmount = String(min);
    bidAuctionId = auction.id;
    selectedAuction = auction;
    showBidConfirm = true;
  }

  function confirmBid() {
    if (!selectedAuction) return;
    const amount = parseInt(bidAmount);
    if (!amount || amount <= 0) return;
    sendWS(ws, 'place_bid', { auctionId: selectedAuction.id, bidAmount: amount });
    showBidConfirm = false;
    selectedAuction = null;
    bidAmount = '';
  }

  function executeBuyout(auction) {
    if (!auction.buyoutPrice) return;
    if (!confirm(`确定以一口价 ${auction.buyoutPrice}💰 购买？`)) return;
    sendWS(ws, 'buyout_auction', { auctionId: auction.id });
  }

  function cancelAuction(auction) {
    if (!confirm('确定要取消此拍卖？商品将退回仓库。')) return;
    sendWS(ws, 'cancel_auction', { auctionId: auction.id });
  }

  function submitCreateAuction() {
    const itemId = createItemId;
    const startingPrice = parseInt(createStartingPrice);
    const buyoutPrice = parseInt(createBuyoutPrice) || 0;

    if (!itemId || !startingPrice || startingPrice <= 0) {
      alert('请选择商品并设定起拍价');
      return;
    }

    if (buyoutPrice > 0 && buyoutPrice <= startingPrice) {
      alert('一口价必须大于起拍价');
      return;
    }

    const listingFee = Math.max(1, Math.floor(startingPrice * 0.05));
    if (listingFee > availableGold) {
      alert(`挂单费 ${listingFee}💰 不足`);
      return;
    }

    sendWS(ws, 'create_auction', { itemId, startingPrice, buyoutPrice });
    showCreateForm = false;
    createItemId = '';
    createStartingPrice = '';
    createBuyoutPrice = '';
  }

  function close() {
    dispatch('close');
  }

  function getQualityRank(q) {
    return { common: 0, fine: 1, rare: 2, legendary: 3 }[q] || 0;
  }
</script>

<div class="modal-overlay" on:click={close}>
  <div class="modal card auction-modal" on:click|stopPropagation>
    <div class="modal-header">
      <h2>🏛️ 拍卖行</h2>
      <div class="header-info">
        <span class="gold-info">💰 {availableGold}</span>
        <span class="frozen-info">❄️ {frozenGold}</span>
      </div>
      <button class="close-btn" on:click={close}>×</button>
    </div>

    <div class="tab-bar">
      <button class="tab-btn" class:active={activeTab === 'browse'} on:click={() => activeTab = 'browse'}>
        浏览竞拍
      </button>
      <button class="tab-btn" class:active={activeTab === 'my-auctions'} on:click={() => activeTab = 'my-auctions'}>
        我的拍卖
      </button>
      <button class="tab-btn" class:active={activeTab === 'my-bids'} on:click={() => activeTab = 'my-bids'}>
        我的竞拍
      </button>
      {#if canListAuction}
        <button class="tab-btn btn-list" on:click={() => showCreateForm = !showCreateForm}>
          {showCreateForm ? '取消挂单' : '📦 挂单出售'}
        </button>
      {/if}
    </div>

    {#if showCreateForm && canListAuction}
      <div class="create-form card">
        <h3>挂单出售</h3>
        <div class="form-row">
          <label>选择商品:</label>
          <select bind:value={createItemId}>
            <option value="">-- 选择仓库商品 --</option>
            {#each warehouseItems as item}
              <option value={item.id}>
                {getItemInfo(item.typeId).name} ({qualityNames[item.quality]})
              </option>
            {/each}
          </select>
        </div>
        <div class="form-row">
          <label>起拍价:</label>
          <input type="number" min="1" bind:value={createStartingPrice} placeholder="最低起拍价" />
          {#if createStartingPrice}
            <span class="fee-hint">挂单费: {Math.max(1, Math.floor(parseInt(createStartingPrice) * 0.05))}💰</span>
          {/if}
        </div>
        <div class="form-row">
          <label>一口价 (可选):</label>
          <input type="number" min="0" bind:value={createBuyoutPrice} placeholder="留空为纯竞拍" />
        </div>
        <button class="btn btn-primary" on:click={submitCreateAuction} disabled={!createItemId || !createStartingPrice}>
          确认挂单
        </button>
      </div>
    {/if}

    <div class="modal-content">
      {#if activeTab === 'browse'}
        <div class="filter-bar">
          <select bind:value={filterCategory}>
            <option value="">全部类别</option>
            <option value="weapon">武器</option>
            <option value="armor">防具</option>
            <option value="consumable">消耗品</option>
            <option value="material">材料</option>
          </select>
          <select bind:value={filterQuality}>
            <option value="">全部品质</option>
            <option value="common">普通</option>
            <option value="fine">精良</option>
            <option value="rare">稀有</option>
            <option value="legendary">传说</option>
          </select>
          <select bind:value={sortBy}>
            {#each sortOptions as opt}
              <option value={opt.value}>{opt.label}</option>
            {/each}
          </select>
        </div>

        {#if filteredAuctions.length === 0}
          <p class="empty">暂无在售商品</p>
        {:else}
          <div class="auction-list">
            {#each filteredAuctions as auction}
              <div class="auction-card quality-{auction.item.quality}">
                <div class="auction-main">
                  <div class="auction-item-info">
                    <span class="item-name quality-{auction.item.quality}">
                      {getItemInfo(auction.item.typeId).name}
                    </span>
                    <span class="item-quality quality-{auction.item.quality}">
                      {qualityNames[auction.item.quality]}
                    </span>
                    <span class="item-category">
                      {categoryNames[getItemInfo(auction.item.typeId).category] || ''}
                    </span>
                  </div>
                  <div class="auction-seller">卖家: {auction.sellerShopName}</div>
                  <div class="auction-prices">
                    <span class="price-label">起拍: {auction.startingPrice}💰</span>
                    <span class="price-current">
                      当前: {auction.currentPrice}💰
                      {#if auction.highestBidderName}
                        <span class="bidder">({auction.highestBidderName})</span>
                      {/if}
                    </span>
                    {#if auction.buyoutPrice > 0}
                      <span class="price-buyout">一口价: {auction.buyoutPrice}💰</span>
                    {/if}
                    <span class="price-remaining">剩余 {getRemainingWeeks(auction, currentWeek)} 周</span>
                  </div>
                </div>
                <div class="auction-actions">
                  {#if auction.sellerId !== playerId}
                    {#if canBid}
                      <button class="btn btn-primary small" on:click={() => openBidConfirm(auction)}>
                        出价
                      </button>
                      {#if auction.buyoutPrice > 0}
                        <button class="btn btn-secondary small" on:click={() => executeBuyout(auction)}
                          disabled={availableGold < auction.buyoutPrice}>
                          一口价
                        </button>
                      {/if}
                    {:else}
                      <span class="hint-small">竞拍在营业/探索日开放</span>
                    {/if}
                  {:else}
                    <span class="own-label">自己的商品</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}

      {:else if activeTab === 'my-auctions'}
        {#if myAuctions.length === 0}
          <p class="empty">你还没有挂出任何商品</p>
        {:else}
          <div class="auction-list">
            {#each myAuctions as auction}
              <div class="auction-card quality-{auction.item.quality}">
                <div class="auction-main">
                  <div class="auction-item-info">
                    <span class="item-name quality-{auction.item.quality}">
                      {getItemInfo(auction.item.typeId).name}
                    </span>
                    <span class="item-quality quality-{auction.item.quality}">
                      {qualityNames[auction.item.quality]}
                    </span>
                    <span class="status-badge status-{auction.status}">
                      {getAuctionStatusText(auction.status)}
                    </span>
                  </div>
                  <div class="auction-prices">
                    <span class="price-label">起拍: {auction.startingPrice}💰</span>
                    <span class="price-current">
                      当前: {auction.currentPrice}💰
                      {#if auction.highestBidderName}
                        <span class="bidder">({auction.highestBidderName})</span>
                      {/if}
                    </span>
                    {#if auction.buyoutPrice > 0}
                      <span class="price-buyout">一口价: {auction.buyoutPrice}💰</span>
                    {/if}
                    {#if auction.status === 'active'}
                      <span class="price-remaining">剩余 {getRemainingWeeks(auction, currentWeek)} 周</span>
                    {/if}
                  </div>
                  {#if auction.bidHistory && auction.bidHistory.length > 0}
                    <div class="bid-history">
                      <span class="bid-count">出价次数: {auction.bidHistory.length}</span>
                    </div>
                  {/if}
                </div>
                <div class="auction-actions">
                  {#if auction.status === 'active' && (!auction.bidHistory || auction.bidHistory.length === 0)}
                    <button class="btn btn-danger small" on:click={() => cancelAuction(auction)}>
                      取消
                    </button>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}

      {:else if activeTab === 'my-bids'}
        {#if myBids.length === 0}
          <p class="empty">你还没有参与任何竞拍</p>
        {:else}
          <div class="auction-list">
            {#each myBids as auction}
              <div class="auction-card quality-{auction.item.quality}">
                <div class="auction-main">
                  <div class="auction-item-info">
                    <span class="item-name quality-{auction.item.quality}">
                      {getItemInfo(auction.item.typeId).name}
                    </span>
                    <span class="item-quality quality-{auction.item.quality}">
                      {qualityNames[auction.item.quality]}
                    </span>
                  </div>
                  <div class="auction-seller">卖家: {auction.sellerShopName}</div>
                  <div class="auction-prices">
                    <span class="price-current">
                      当前价: {auction.currentPrice}💰
                    </span>
                    {#if auction.highestBidderId === playerId}
                      <span class="bid-status leading">🏆 你的出价领先</span>
                    {:else}
                      <span class="bid-status outbid">⚠️ 你已被超过</span>
                    {/if}
                    <span class="price-remaining">剩余 {getRemainingWeeks(auction, currentWeek)} 周</span>
                  </div>
                </div>
                <div class="auction-actions">
                  {#if canBid}
                    <button class="btn btn-primary small" on:click={() => openBidConfirm(auction)}>
                      再次出价
                    </button>
                    {#if auction.buyoutPrice > 0}
                      <button class="btn btn-secondary small" on:click={() => executeBuyout(auction)}
                        disabled={availableGold < auction.buyoutPrice}>
                        一口价
                      </button>
                    {/if}
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      {/if}
    </div>

    {#if showBidConfirm && selectedAuction}
      <div class="bid-confirm-overlay" on:click={() => showBidConfirm = false}>
        <div class="bid-confirm card" on:click|stopPropagation>
          <h3>确认出价</h3>
          <div class="confirm-info">
            <p>商品: <span class="quality-{selectedAuction.item.quality}">{getItemInfo(selectedAuction.item.typeId).name} ({qualityNames[selectedAuction.item.quality]})</span></p>
            <p>当前价: <strong>{selectedAuction.currentPrice}💰</strong></p>
            <p>最低出价: <strong>{getMinBid(selectedAuction)}💰</strong></p>
            <p>可用余额: <strong>{availableGold}💰</strong></p>
          </div>
          <div class="bid-input">
            <input type="number" min={getMinBid(selectedAuction)} bind:value={bidAmount} />
            <span>💰</span>
          </div>
          <div class="confirm-actions">
            <button class="btn btn-primary" on:click={confirmBid}
              disabled={!bidAmount || parseInt(bidAmount) < getMinBid(selectedAuction) || parseInt(bidAmount) > availableGold}>
              确认出价
            </button>
            <button class="btn btn-secondary" on:click={() => showBidConfirm = false}>取消</button>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .auction-modal {
    width: 95%;
    max-width: 800px;
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
  }

  .modal-header h2 {
    color: var(--primary);
  }

  .header-info {
    display: flex;
    gap: 15px;
    align-items: center;
  }

  .gold-info {
    font-weight: bold;
    color: var(--secondary);
  }

  .frozen-info {
    font-weight: bold;
    color: #60a5fa;
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

  .tab-btn:hover:not(.active) {
    background: rgba(139, 92, 246, 0.2);
  }

  .btn-list {
    margin-left: auto;
    background: linear-gradient(135deg, var(--secondary) 0%, #d97706 100%);
    border-color: var(--secondary);
    color: white;
  }

  .create-form {
    padding: 15px;
    margin-bottom: 15px;
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid var(--secondary);
  }

  .create-form h3 {
    color: var(--secondary);
    margin-bottom: 12px;
  }

  .form-row {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 10px;
  }

  .form-row label {
    min-width: 100px;
    font-size: 14px;
  }

  .form-row select,
  .form-row input {
    flex: 1;
  }

  .fee-hint {
    font-size: 12px;
    color: var(--secondary);
    white-space: nowrap;
  }

  .modal-content {
    flex: 1;
    overflow-y: auto;
  }

  .filter-bar {
    display: flex;
    gap: 10px;
    margin-bottom: 15px;
  }

  .filter-bar select {
    flex: 1;
    padding: 8px 10px;
    font-size: 13px;
  }

  .auction-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .auction-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 15px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 8px;
    border-left: 4px solid var(--primary);
    gap: 12px;
  }

  .auction-card.quality-fine { border-left-color: #22c55e; }
  .auction-card.quality-rare { border-left-color: #3b82f6; }
  .auction-card.quality-legendary { border-left-color: #f59e0b; }

  .auction-main {
    flex: 1;
  }

  .auction-item-info {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 6px;
  }

  .item-name {
    font-weight: 600;
    font-size: 15px;
  }

  .item-quality {
    font-size: 12px;
  }

  .item-category {
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 10px;
    background: rgba(139, 92, 246, 0.2);
  }

  .auction-seller {
    font-size: 12px;
    color: var(--gray);
    margin-bottom: 4px;
  }

  .auction-prices {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    font-size: 13px;
  }

  .price-label {
    color: var(--gray);
  }

  .price-current {
    font-weight: 600;
    color: var(--light);
  }

  .price-buyout {
    color: var(--secondary);
    font-weight: 600;
  }

  .price-remaining {
    color: #60a5fa;
  }

  .bidder {
    color: var(--gray);
    font-size: 12px;
  }

  .bid-history {
    margin-top: 4px;
  }

  .bid-count {
    font-size: 12px;
    color: var(--gray);
  }

  .auction-actions {
    display: flex;
    flex-direction: column;
    gap: 6px;
    align-items: flex-end;
  }

  .btn.small {
    padding: 6px 12px;
    font-size: 12px;
    white-space: nowrap;
  }

  .own-label {
    font-size: 12px;
    color: var(--gray);
    font-style: italic;
  }

  .hint-small {
    font-size: 11px;
    color: var(--gray);
  }

  .status-badge {
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 10px;
    font-weight: 600;
  }

  .status-active {
    background: rgba(16, 185, 129, 0.2);
    color: var(--success);
  }

  .status-sold {
    background: rgba(139, 92, 246, 0.2);
    color: var(--primary);
  }

  .status-expired {
    background: rgba(239, 68, 68, 0.2);
    color: var(--danger);
  }

  .status-cancelled {
    background: rgba(100, 116, 139, 0.2);
    color: var(--gray);
  }

  .bid-status {
    font-size: 12px;
    font-weight: 600;
  }

  .bid-status.leading {
    color: var(--success);
  }

  .bid-status.outbid {
    color: var(--danger);
  }

  .empty {
    text-align: center;
    color: var(--gray);
    padding: 40px;
  }

  .bid-confirm-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10;
    border-radius: 12px;
  }

  .bid-confirm {
    padding: 20px;
    max-width: 350px;
    width: 90%;
  }

  .bid-confirm h3 {
    color: var(--primary);
    margin-bottom: 15px;
  }

  .confirm-info {
    margin-bottom: 15px;
    font-size: 14px;
  }

  .confirm-info p {
    margin-bottom: 6px;
  }

  .bid-input {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 15px;
  }

  .bid-input input {
    flex: 1;
    padding: 10px;
    font-size: 16px;
  }

  .confirm-actions {
    display: flex;
    gap: 10px;
  }

  .confirm-actions .btn {
    flex: 1;
  }
</style>
