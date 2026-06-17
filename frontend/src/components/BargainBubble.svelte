<script>
  import { onMount, onDestroy } from 'svelte';
  import { bargainRequests, removeBargainRequest } from '../stores/gameStore.js';
  import { sendWS } from '../utils/api.js';

  export let ws;

  const qualityNames = {
    common: '普通',
    fine: '精良',
    rare: '稀有',
    legendary: '传说',
  };

  const classIcons = {
    warrior: '⚔️',
    mage: '🔮',
    rogue: '🗡️',
  };

  const classNames = {
    warrior: '战士',
    mage: '法师',
    rogue: '盗贼',
  };

  let countdowns = {};
  let intervals = {};

  $: activeRequests = $bargainRequests;

  function startCountdown(bargain) {
    const now = Math.floor(Date.now() / 1000);
    const total = 5;
    const remaining = Math.max(0, bargain.expiresAt - now);
    countdowns[bargain.id] = {
      remaining,
      total,
      percent: Math.max(0, (remaining / total) * 100),
    };

    if (intervals[bargain.id]) clearInterval(intervals[bargain.id]);
    intervals[bargain.id] = setInterval(() => {
      const cur = Math.floor(Date.now() / 1000);
      const rem = Math.max(0, bargain.expiresAt - cur);
      countdowns[bargain.id] = {
        remaining: rem,
        total,
        percent: Math.max(0, (rem / total) * 100),
      };
      if (rem <= 0) {
        clearInterval(intervals[bargain.id]);
        delete intervals[bargain.id];
      }
    }, 100);
  }

  function acceptBargain(bargain) {
    sendWS(ws, 'bargain_accept', { bargainId: bargain.id });
    removeBargainRequest(bargain.id);
    if (intervals[bargain.id]) {
      clearInterval(intervals[bargain.id]);
      delete intervals[bargain.id];
    }
    delete countdowns[bargain.id];
  }

  function rejectBargain(bargain) {
    sendWS(ws, 'bargain_reject', { bargainId: bargain.id });
    removeBargainRequest(bargain.id);
    if (intervals[bargain.id]) {
      clearInterval(intervals[bargain.id]);
      delete intervals[bargain.id];
    }
    delete countdowns[bargain.id];
  }

  function getDiscountPct(bargain) {
    const diff = bargain.originalPrice - bargain.bargainedPrice;
    const pct = ((diff / bargain.originalPrice) * 100).toFixed(0);
    return pct;
  }

  function ensureCountdown(bargain) {
    if (!countdowns[bargain.id] || countdowns[bargain.id].remaining > 0) {
      if (!intervals[bargain.id]) {
        startCountdown(bargain);
      }
    }
    return countdowns[bargain.id] || { remaining: 0, total: 5, percent: 0 };
  }

  onDestroy(() => {
    Object.keys(intervals).forEach(id => clearInterval(intervals[id]));
  });
</script>

<div class="bargain-container">
  {#each activeRequests as bargain (bargain.id)}
    {@const cd = ensureCountdown(bargain)}
    {@const cdPct = cd.percent}
    <div class="bargain-bubble quality-{bargain.itemQuality}">
      <div class="bubble-header">
        <div class="npc-info">
          <span class="npc-icon">{classIcons[bargain.npcClass] || '👤'}</span>
          <div>
            <div class="npc-name">{bargain.npcName}</div>
            <div class="npc-class">{classNames[bargain.npcClass] || '冒险者'}</div>
          </div>
        </div>
        <div class="bargain-tag">砍价请求</div>
      </div>

      <div class="bubble-body">
        <div class="item-info">
          <span class="item-quality quality-{bargain.itemQuality}">◆</span>
          <span class="item-name">{bargain.itemName}</span>
          <span class="item-quality-tag">[{qualityNames[bargain.itemQuality]}]</span>
        </div>

        <div class="price-compare">
          <div class="price-row original">
            <span class="price-label">原价</span>
            <span class="price-value strike">{bargain.originalPrice} 💰</span>
          </div>
          <div class="arrow">⬇️ -{getDiscountPct(bargain)}%</div>
          <div class="price-row bargained">
            <span class="price-label">砍后价</span>
            <span class="price-value highlight">{bargain.bargainedPrice} 💰</span>
          </div>
        </div>

        <div class="countdown-wrap">
          <div class="countdown-bar">
            <div
              class="countdown-fill"
              style="width: {cdPct}%;"
              class:urgent={cdPct < 40}
            ></div>
          </div>
          <div class="countdown-text">
            ⏱️ {cd.remaining}s 后自动拒绝
          </div>
        </div>
      </div>

      <div class="bubble-actions">
        <button class="btn btn-danger" on:click={() => rejectBargain(bargain)}>
          ❌ 拒绝
        </button>
        <button class="btn btn-success" on:click={() => acceptBargain(bargain)}>
          ✅ 同意降价
        </button>
      </div>
    </div>
  {/each}
</div>

<style>
  .bargain-container {
    position: fixed;
    top: 80px;
    right: 20px;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    gap: 12px;
    max-width: 340px;
    pointer-events: none;
  }

  .bargain-bubble {
    pointer-events: auto;
    background: var(--card-bg);
    border: 2px solid var(--primary);
    border-radius: 16px;
    padding: 14px;
    box-shadow: 0 8px 32px rgba(139, 92, 246, 0.35);
    animation: slideIn 0.3s cubic-bezier(0.2, 0.9, 0.3, 1.2);
  }

  @keyframes slideIn {
    from {
      transform: translateX(100%);
      opacity: 0;
    }
    to {
      transform: translateX(0);
      opacity: 1;
    }
  }

  .bargain-bubble.quality-fine {
    border-color: #3b82f6;
    box-shadow: 0 8px 32px rgba(59, 130, 246, 0.35);
  }
  .bargain-bubble.quality-rare {
    border-color: #a855f7;
    box-shadow: 0 8px 32px rgba(168, 85, 247, 0.35);
  }
  .bargain-bubble.quality-legendary {
    border-color: #f59e0b;
    box-shadow: 0 8px 32px rgba(245, 158, 11, 0.4);
    animation: slideIn 0.3s cubic-bezier(0.2, 0.9, 0.3, 1.2),
      legendaryGlow 2s ease-in-out infinite alternate;
  }

  @keyframes legendaryGlow {
    from {
      box-shadow: 0 8px 32px rgba(245, 158, 11, 0.4);
    }
    to {
      box-shadow: 0 8px 48px rgba(245, 158, 11, 0.7);
    }
  }

  .bubble-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
    padding-bottom: 8px;
    border-bottom: 1px solid rgba(139, 92, 246, 0.2);
  }

  .npc-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .npc-icon {
    font-size: 28px;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: rgba(139, 92, 246, 0.15);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .npc-name {
    font-weight: 700;
    font-size: 15px;
  }

  .npc-class {
    font-size: 11px;
    color: var(--gray);
  }

  .bargain-tag {
    background: linear-gradient(135deg, #f59e0b, #ef4444);
    color: white;
    padding: 3px 10px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 600;
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.7; }
  }

  .bubble-body {
    margin-bottom: 12px;
  }

  .item-info {
    text-align: center;
    margin-bottom: 10px;
    font-weight: 600;
  }

  .item-name {
    margin: 0 4px;
  }

  .item-quality-tag {
    font-size: 11px;
    color: var(--gray);
  }

  .quality-common { color: #9ca3af; }
  .quality-fine { color: #3b82f6; }
  .quality-rare { color: #a855f7; }
  .quality-legendary { color: #f59e0b; }

  .price-compare {
    background: rgba(0, 0, 0, 0.2);
    border-radius: 10px;
    padding: 10px;
    margin-bottom: 10px;
  }

  .price-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 2px 0;
  }

  .price-label {
    font-size: 12px;
    color: var(--gray);
  }

  .price-value {
    font-weight: 700;
    font-size: 15px;
  }

  .price-value.strike {
    text-decoration: line-through;
    color: #9ca3af;
    font-size: 13px;
  }

  .price-value.highlight {
    color: #10b981;
    font-size: 18px;
  }

  .arrow {
    text-align: center;
    font-size: 12px;
    color: #ef4444;
    font-weight: 600;
    padding: 2px 0;
  }

  .countdown-wrap {
    margin-top: 8px;
  }

  .countdown-bar {
    width: 100%;
    height: 8px;
    background: rgba(0, 0, 0, 0.3);
    border-radius: 4px;
    overflow: hidden;
    margin-bottom: 4px;
  }

  .countdown-fill {
    height: 100%;
    background: linear-gradient(90deg, #10b981, #3b82f6);
    border-radius: 4px;
    transition: width 0.1s linear;
  }

  .countdown-fill.urgent {
    background: linear-gradient(90deg, #f59e0b, #ef4444);
    animation: urgentPulse 0.4s ease-in-out infinite alternate;
  }

  @keyframes urgentPulse {
    from { opacity: 0.8; }
    to { opacity: 1; }
  }

  .countdown-text {
    text-align: center;
    font-size: 11px;
    color: var(--gray);
  }

  .bubble-actions {
    display: flex;
    gap: 8px;
  }

  .btn {
    flex: 1;
    padding: 10px;
    border: none;
    border-radius: 10px;
    font-weight: 600;
    font-size: 13px;
    cursor: pointer;
    transition: transform 0.15s, filter 0.15s;
  }

  .btn:hover {
    transform: translateY(-1px);
    filter: brightness(1.1);
  }

  .btn:active {
    transform: translateY(0);
  }

  .btn-success {
    background: linear-gradient(135deg, #10b981, #059669);
    color: white;
  }

  .btn-danger {
    background: linear-gradient(135deg, #6b7280, #4b5563);
    color: white;
  }
</style>
