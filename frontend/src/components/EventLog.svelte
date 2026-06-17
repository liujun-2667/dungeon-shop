<script>
  import { eventLog } from '../stores/gameStore.js';
  import { onMount, onDestroy } from 'svelte';

  const typeColors = {
    info: 'var(--primary)',
    success: 'var(--success)',
    error: 'var(--danger)',
    warning: 'var(--secondary)',
    chat: 'var(--gray)',
  };

  let displayedLogs = [];
  let logQueue = [];
  let logContainer = null;
  let queueTimer = null;
  let lastLogCount = 0;

  $: {
    const allLogs = $eventLog;
    if (allLogs.length > lastLogCount) {
      const newLogs = allLogs.slice(lastLogCount);
      logQueue = [...logQueue, ...newLogs];
      lastLogCount = allLogs.length;
    } else if (allLogs.length < lastLogCount) {
      lastLogCount = allLogs.length;
      displayedLogs = allLogs.slice();
      logQueue = [];
    }
  }

  function processQueue() {
    if (logQueue.length > 0) {
      const [next, ...rest] = logQueue;
      logQueue = rest;
      displayedLogs = [...displayedLogs, next];
    }
  }

  $: displayedLogs, scrollToBottom()

  function scrollToBottom() {
    if (logContainer) {
      requestAnimationFrame(() => {
        if (logContainer) {
          logContainer.scrollTop = logContainer.scrollHeight;
        }
      });
    }
  }

  onMount(() => {
    displayedLogs = $eventLog.slice();
    lastLogCount = $eventLog.length;
    queueTimer = setInterval(processQueue, 300);
  });

  onDestroy(() => {
    if (queueTimer) {
      clearInterval(queueTimer);
    }
  });
</script>

<div class="event-log">
  <div class="log-header">
    <h4>📜 事件日志</h4>
  </div>
  <div class="log-content" bind:this={logContainer}>
    {#if displayedLogs.length === 0}
      <p class="empty">暂无事件</p>
    {:else}
      {#each displayedLogs as log (log.id)}
        <div class="log-item" style="border-left-color: {typeColors[log.type]}">
          <span class="log-time">{log.time}</span>
          <span class="log-message">{log.message}</span>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .event-log {
    margin-top: 10px;
    background: var(--card-bg);
    border-radius: 12px;
    border: 1px solid var(--border);
    max-height: 150px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .log-header {
    padding: 10px 15px;
    border-bottom: 1px solid var(--border);
  }

  .log-header h4 {
    color: var(--primary);
    font-size: 14px;
  }

  .log-content {
    flex: 1;
    overflow-y: auto;
    padding: 10px;
    display: flex;
    flex-direction: column;
  }

  .log-item {
    display: flex;
    gap: 10px;
    padding: 6px 10px;
    margin-bottom: 4px;
    border-left: 3px solid var(--primary);
    background: rgba(0, 0, 0, 0.2);
    border-radius: 0 4px 4px 0;
    font-size: 12px;
    animation: fadeInSlide 0.3s ease-out;
  }

  @keyframes fadeInSlide {
    from {
      opacity: 0;
      transform: translateY(-10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .log-time {
    color: var(--gray);
    flex-shrink: 0;
  }

  .log-message {
    flex: 1;
  }

  .empty {
    text-align: center;
    color: var(--gray);
    padding: 20px;
    font-size: 12px;
    margin: auto;
  }
</style>
