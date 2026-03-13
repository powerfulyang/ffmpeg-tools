<template>
  <div class="card" v-if="!ready">
    <div class="ffmpeg-status">
      <div v-if="error" class="status error">
        <span>❌ FFmpeg 初始化失败: {{ error }}</span>
      </div>
      <div v-else class="ffmpeg-loading">
        <span class="loader"></span>
        <div class="ffmpeg-info">
          <p class="ffmpeg-status-text">{{ status || '正在检查 FFmpeg...' }}</p>
          <div class="progress-bar" v-if="progress > 0">
            <div class="progress-fill" :style="{ width: progress + '%' }"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue'

export default defineComponent({
  name: 'FFmpegStatus',
  props: {
    ready: {
      type: Boolean,
      default: false
    },
    status: {
      type: String,
      default: ''
    },
    progress: {
      type: Number,
      default: 0
    },
    error: {
      type: String,
      default: ''
    }
  }
})
</script>

<style scoped>
.ffmpeg-status {
  padding: 24px;
}

.ffmpeg-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.ffmpeg-info {
  flex: 1;
}

.ffmpeg-status-text {
  color: var(--text-primary);
  margin-bottom: 8px;
}

.loader {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border-color);
  border-top-color: #00d4aa;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.progress-bar {
  height: 6px;
  background: var(--border-color);
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #00d4aa, #00b894);
  transition: width 0.3s ease;
}

.status {
  padding: 12px 16px;
  border-radius: 8px;
  font-size: 0.9rem;
}

.status.error {
  background: rgba(255, 71, 87, 0.1);
  color: #ff4757;
}
</style>
