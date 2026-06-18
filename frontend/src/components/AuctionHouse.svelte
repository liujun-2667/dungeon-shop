<script>
  import { createEventDispatcher } from 'svelte';
  import {
    room,
    currentUser,
    currentPlayer,
    currentPhase,
    itemTypes,
    guilds,
    addLog,
    getActiveAuctions,
    getActiveGuildAuctions,
    getMyAuctions,
    getMyBids,
    getMinBid,
    getRemainingWeeks,
    getAuctionStatusText,
    getAuctionReputationColor,
    getAuctionListingFeeRate,
    getAuctionListingFeeTier,
    canListAuction,
    getPlayerAuctionReputation,
    getMyAuctionHistory,
    getMyBidHistory,
    getCurrentGuild,
  } from '../stores/gameStore.js';
  import { sendWS } from '../utils/api.js';

  const dispatch = createEventDispatcher();

  export let ws;

  let activeTab = 'browse';
  let historySubTab = 'active';
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
  let createIsGuildAuction = false;

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
  $: playerGuildId = $currentPlayer?.guildId || '';
  $: currentGuild = getCurrentGuild($guilds, playerGuildId);
  $: activeAuctions = getActiveAuctions($room);
  $: activeGuildAuctions = getActiveGuildAuctions($room, playerGuildId);
  $: myAuctions = getMyAuctions($room, playerId);
  $: myBids = getMyBids($room, playerId);
  $: myAuctionHistory = getMyAuctionHistory($room, playerId);
  $: myBidHistory = getMyBidHistory($room, playerId);
  $: availableGold = $currentPlayer?.gold || 0;
  $: frozenGold = $currentPlayer?.frozenGold || 0;
  $: frozenDeposit = $currentPlayer?.frozenDeposit || 0;
  $: playerAuctionReputation = $currentPlayer?.auctionReputation ?? 100;
  $: listingFeeRate = getAuctionListingFeeRate(playerAuctionReputation);
  $: listingFeeTier = getAuctionListingFeeTier(playerAuctionReputation);

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

  $: canListAuctionFlag = $currentPhase === 'purchase' && !$currentPlayer?.isBankrupt && canListAuction(playerAuctionReputation);
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

    if (!createIsGuildAuction) {
      const listingFee = Math.max(1, Math.floor(startingPrice * listingFeeRate));
      if (listingFee > availableGold) {
        alert(`挂单费 ${listingFee}💰 不足`);
        return;
      }
      sendWS(ws, 'create_auction', { itemId, startingPrice, buyoutPrice });
    } else {
      sendWS(ws, 'create_guild_auction', { itemId, startingPrice, buyoutPrice });
    }

    showCreateForm = false;
    createItemId = '';
    createStartingPrice = '';
    createBuyoutPrice = '';
    createIsGuildAuction = false;
  }

  function getBidDeposit(amount) {
    return Math.floor(amount * 0.10);
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
        {#if frozenDeposit > 0}
          <span class="deposit-info">🔒 {frozenDeposit}</span>
        {/if}
        <span class="rep-info" style="color: {getAuctionReputationColor(playerAuctionReputation)}">
          🏛️ {playerAuctionReputation}
        </span>
      </div>
      <button class="close-btn" on:click={close}>×</button>
    </div>

    <div class="tab-bar">
      <button class="tab-btn" class:active={activeTab === 'browse'} on:click={() => activeTab = 'browse'}>
        浏览竞拍
      </button>
      {#if currentGuild}
        <button class="tab-btn" class:active={activeTab === 'guild-auctions'} on:click={() => activeTab = 'guild-auctions'}>
          🏰 公会拍卖
        </button>
      {/if}
      <button class="tab-btn" class:active={activeTab === 'my-auctions'} on:click={() => activeTab = 'my-auctions'}>
        我的拍卖
      </button>
      <button class="tab-btn" class:active={activeTab === 'my-bids'} on:click={() => activeTab = 'my-bids'}>
        我的竞拍
      </button>
      {#if canListAuctionFlag || currentGuild}
        <button class="tab-btn btn-list" on:click={() => showCreateForm = !showCreateForm}>
          {showCreateForm ? '取消挂单' : '📦 挂单出售'}
        </button>
      {/if}
    </div>

    {#if $currentPhase === 'purchase' && !$currentPlayer?.isBankrupt && !canListAuction(playerAuctionReputation)}
      <div class="create-form card forbid-card">
        <h3>⚠️ 无法挂单</h3>
        <p class="forbid-hint">拍卖行信誉分不足40分（当前 {playerAuctionReputation}），禁止挂单拍卖，仍可参与竞拍。</p>
      </div>
    {/if}
    {#if showCreateForm && (canListAuctionFlag || currentGuild)}
      <div class="create-form card">
        <h3>挂单出售
          {#if !createIsGuildAuction}
            <span class="fee-tier tier-{listingFeeTier}">
              费率 {(listingFeeRate * 100).toFixed(0)}%
              ({listingFeeTier === 'honest' ? '优惠' : listingFeeTier === 'shady' ? '惩罚' : '标准'})
            </span>
          {/if}
          {#if createIsGuildAuction}
            <span class="fee-tier tier-honest">
              🏰 公会内部拍卖（免挂单费，不计信誉）
            </span>
          {/if}
        </h3>
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
          {#if createStartingPrice && !createIsGuildAuction}
            <span class="fee-hint">挂单费: {Math.max(1, Math.floor(parseInt(createStartingPrice) * listingFeeRate))}💰</span>
          {/if}
          {#if createStartingPrice && createIsGuildAuction}
            <span class="fee-hint free">挂单费: 免费</span>
          {/if}
        </div>
        <div class="form-row">
          <label>一口价 (可选):</label>
          <input type="number" min="0" bind:value={createBuyoutPrice} placeholder="留空为纯竞拍" />
        </div>
        {#if currentGuild}
          <div class="form-row">
            <label>
              <input type="checkbox" bind:checked={createIsGuildAuction} />
              设为公会内部拍卖（仅公会成员可见，免挂单费）
            </label>
          </div>
        {/if}
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
                  <div class="auction-seller">
                    卖家: {auction.sellerShopName}
                    <span class="seller-reputation" 
                          style="color: {getAuctionReputationColor(getPlayerAuctionReputation($room, auction.sellerId))}">
                      🏛️ {getPlayerAuctionReputation($room, auction.sellerId)}
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

      {:else if activeTab === 'guild-auctions'}
        <div class="guild-auction-header">
          <h4>🏰 公会内部拍卖 - [{currentGuild.abbreviation}] {currentGuild.name}</h4>
          <p class="hint">仅公会成员可见，免挂单费，流拍不扣信誉</p>
        </div>
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

        {#if activeGuildAuctions.length === 0}
          <p class="empty">暂无公会内部拍卖</p>
        {:else}
          <div class="auction-list">
            {#each activeGuildAuctions as auction}
              <div class="auction-card quality-{auction.item.quality} guild-auction-card">
                <div class="guild-badge">🏰 公会拍卖</div>
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
                  <div class="auction-seller">
                    卖家: {auction.sellerShopName}
                    <span class="seller-reputation" 
                          style="color: {getAuctionReputationColor(getPlayerAuctionReputation($room, auction.sellerId))}">
                      🏛️ {getPlayerAuctionReputation($room, auction.sellerId)}
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
                    <button class="btn btn-danger small" on:click={() => cancelAuction(auction)}>
                      取消拍卖
                    </button>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}

      {:else if activeTab === 'my-auctions'}
        <div class="sub-tab-bar">
          <button class="sub-tab-btn" class:active={historySubTab === 'active'} on:click={() => historySubTab = 'active'}>
            进行中
          </button>
          <button class="sub-tab-btn" class:active={historySubTab === 'history'} on:click={() => historySubTab = 'history'}>
            历史记录
          </button>
        </div>
        {#if historySubTab === 'active'}
          {#if myAuctions.filter(a => a.status === 'active').length === 0}
            <p class="empty">暂无进行中的拍卖</p>
          {:else}
            <div class="auction-list">
              {#each myAuctions.filter(a => a.status === 'active') as auction}
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
                      <span class="price-remaining">剩余 {getRemainingWeeks(auction, currentWeek)} 周</span>
                    </div>
                    {#if auction.bidHistory && auction.bidHistory.length > 0}
                      <div class="bid-history">
                        <span class="bid-count">出价次数: {auction.bidHistory.length}</span>
                      </div>
                    {/if}
                  </div>
                  <div class="auction-actions">
                    {#if !auction.bidHistory || auction.bidHistory.length === 0}
                      <button class="btn btn-danger small" on:click={() => cancelAuction(auction)}>
                        取消
                      </button>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        {:else}
          {#if myAuctionHistory.length === 0}
            <p class="empty">暂无历史拍卖记录</p>
          {:else}
            <div class="auction-list">
              {#each myAuctionHistory as record}
                <div class="auction-card history-card quality-{record.itemQuality}">
                  <div class="auction-main">
                    <div class="auction-item-info">
                      <span class="item-name quality-{record.itemQuality}">
                        {record.itemTypeName}
                      </span>
                      <span class="item-quality quality-{record.itemQuality}">
                        {qualityNames[record.itemQuality]}
                      </span>
                      <span class="status-badge status-{record.status}">
                        {getAuctionStatusText(record.status)}
                      </span>
                    </div>
                    <div class="auction-prices">
                      <span class="price-current">成交价: {record.finalPrice}💰</span>
                      <span class="price-buyer">买家: {record.buyer}</span>
                    </div>
                    <div class="rep-change-row">
                      信誉变动: 
                      <span class="rep-change" style="color: {record.repChange >= 0 ? '#10b981' : '#ef4444'}">
                        {record.repChange > 0 ? '+' : ''}{record.repChange}
                      </span>
                    </div>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        {/if}

      {:else if activeTab === 'my-bids'}
        <div class="sub-tab-bar">
          <button class="sub-tab-btn" class:active={historySubTab === 'active'} on:click={() => historySubTab = 'active'}>
            进行中
          </button>
          <button class="sub-tab-btn" class:active={historySubTab === 'history'} on:click={() => historySubTab = 'history'}>
            历史记录
          </button>
        </div>
        {#if historySubTab === 'active'}
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
        {:else}
          {#if myBidHistory.length === 0}
            <p class="empty">暂无历史竞拍记录</p>
          {:else}
            <div class="auction-list">
              {#each myBidHistory as record}
                <div class="auction-card history-card quality-{record.itemQuality}">
                  <div class="auction-main">
                    <div class="auction-item-info">
                      <span class="item-name quality-{record.itemQuality}">
                        {record.itemTypeName}
                      </span>
                      <span class="item-quality quality-{record.itemQuality}">
                        {qualityNames[record.itemQuality]}
                      </span>
                      <span class="status-badge status-{record.status}">
                        {getAuctionStatusText(record.status)}
                      </span>
                    </div>
                    <div class="auction-prices">
                      <span class="price-label">我的最高出价: {record.myMaxBid}💰</span>
                      <span class="price-current">成交价: {record.finalPrice}💰</span>
                      {#if record.won}
                        <span class="bid-status leading">🏆 已竞得</span>
                      {:else if record.status === 'sold'}
                        <span class="bid-status outbid">未竞得</span>
                      {:else}
                        <span class="bid-status outbid">{record.status === 'expired' ? '已流拍' : '已取消'}</span>
                      {/if}
                    </div>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
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
            {#if bidAmount}
              <p class="deposit-hint">
                保证金(10%): <strong>{getBidDeposit(parseInt(bidAmount) || 0)}💰</strong>
                <br/>
                <span class="deposit-total">总计需支付: {parseInt(bidAmount) + getBidDeposit(parseInt(bidAmount) || 0)}💰</span>
              </p>
            {/if}
          </div>
          <div class="bid-input">
            <input type="number" min={getMinBid(selectedAuction)} bind:value={bidAmount} />
            <span>💰</span>
          </div>
          <div class="confirm-actions">
            <button class="btn btn-primary" on:click={confirmBid}
              disabled={!bidAmount || parseInt(bidAmount) < getMinBid(selectedAuction) || (parseInt(bidAmount) + getBidDeposit(parseInt(bidAmount) || 0)) > availableGold}>
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

  .deposit-info {
    font-weight: bold;
    color: #f59e0b;
  }

  .rep-info {
    font-weight: bold;
    padding: 2px 8px;
    border-radius: 6px;
    background: rgba(0, 0, 0, 0.3);
    font-size: 13px;
  }

  .fee-tier {
    display: inline-block;
    padding: 3px 10px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 600;
    margin-left: 10px;
    vertical-align: middle;
  }

  .fee-tier.tier-honest {
    background: linear-gradient(135deg, #10b981, #059669);
    color: white;
  }

  .fee-tier.tier-normal {
    background: linear-gradient(135deg, #6366f1, #4f46e5);
    color: white;
  }

  .fee-tier.tier-shady {
    background: linear-gradient(135deg, #f59e0b, #d97706);
    color: white;
  }

  .fee-tier.tier-forbid {
    background: linear-gradient(135deg, #ef4444, #dc2626);
    color: white;
  }

  .forbid-card {
    border: 1px solid var(--danger);
    background: rgba(239, 68, 68, 0.1);
  }

  .forbid-card h3 {
    color: var(--danger);
  }

  .forbid-hint {
    font-size: 14px;
    color: var(--light);
  }

  .sub-tab-bar {
    display: flex;
    gap: 8px;
    margin-bottom: 15px;
  }

  .sub-tab-btn {
    padding: 6px 14px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: rgba(0, 0, 0, 0.2);
    color: var(--gray);
    cursor: pointer;
    font-size: 13px;
    font-weight: 500;
    transition: all 0.2s;
  }

  .sub-tab-btn.active {
    background: var(--primary);
    border-color: var(--primary);
    color: white;
  }

  .sub-tab-btn:hover:not(.active) {
    background: rgba(139, 92, 246, 0.15);
    color: var(--light);
  }

  .seller-reputation {
    font-size: 12px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
    background: rgba(0, 0, 0, 0.2);
    margin-left: 8px;
  }

  .history-card {
    opacity: 0.9;
    border-left-color: var(--gray) !important;
  }

  .rep-change-row {
    font-size: 13px;
    margin-top: 4px;
    color: var(--gray);
  }

  .rep-change {
    font-weight: 700;
    padding: 1px 6px;
    border-radius: 4px;
    background: rgba(0, 0, 0, 0.2);
    margin-left: 4px;
  }

  .price-buyer {
    font-size: 13px;
    color: var(--primary);
    font-weight: 500;
  }

  .deposit-hint {
    font-size: 13px;
    color: var(--secondary);
    padding: 8px;
    background: rgba(245, 158, 11, 0.1);
    border-radius: 6px;
    margin-top: 8px;
  }

  .deposit-total {
    display: inline-block;
    margin-top: 4px;
    color: var(--danger);
    font-weight: 700;
  }

  .guild-auction-header {
    background: linear-gradient(135deg, rgba(16, 185, 129, 0.1), rgba(5, 150, 105, 0.1));
    border: 1px solid rgba(16, 185, 129, 0.3);
    border-radius: 10px;
    padding: 15px;
    margin-bottom: 15px;
  }

  .guild-auction-header h4 {
    margin: 0 0 5px;
    color: var(--success);
  }

  .guild-auction-header .hint {
    margin: 0;
    color: var(--gray);
    font-size: 13px;
  }

  .guild-auction-card {
    border-color: rgba(16, 185, 129, 0.3);
    background: rgba(16, 185, 129, 0.05);
    position: relative;
  }

  .guild-badge {
    position: absolute;
    top: 10px;
    right: 10px;
    background: linear-gradient(135deg, #10b981, #059669);
    color: white;
    padding: 3px 10px;
    border-radius: 6px;
    font-size: 11px;
    font-weight: 600;
  }

  .fee-hint.free {
    color: var(--success);
    font-weight: 600;
  }
</style>
