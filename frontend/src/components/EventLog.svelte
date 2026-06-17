<script>
  import { eventLog } from '../stores/gameStore.js';

  const typeColors = {
    info: 'var(--primary)',
    success: 'var(--success)',
    error: 'var(--danger)',
    warning: 'var(--secondary)',
    chat: 'var(--gray)',
  };
</script>

<div class="event-log">
  <div class="log-header">
    <h4>📜 事件日志</h4>
  </div>
  <div class="log-content">
    {#each $eventLog as log}
      <div class="log-item" style="border-left-color: {typeColors[log.type]}">
        <span class="log-time">{log.time}</span>
        <span class="log-message">{log.message}</span>
      </div>
    {/each}
    {#if $eventLog.length === 0}
      <p class="empty">暂无事件</p>
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
  }
</style>
