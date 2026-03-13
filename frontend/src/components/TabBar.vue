<template>
  <div class="card tab-bar">
    <button
      class="tab-btn"
      :class="{ active: activeTab === 'mov2webm' }"
      @click="$emit('change', 'mov2webm')"
    >
      🔄 MOV 转 WebM
    </button>
    <button
      class="tab-btn"
      :class="{ active: activeTab === 'ppt' }"
      @click="$emit('change', 'ppt')"
    >
      📊 PPT 视频压缩
    </button>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue'

export default defineComponent({
  name: 'TabBar',
  props: {
    activeTab: {
      type: String,
      default: 'mov2webm'
    }
  },
  emits: ['change']
})
</script>

<style scoped>
.tab-bar {
  display: flex;
  padding: 0;
  gap: 0;
  margin-bottom: 20px;
}

.tab-btn {
  flex: 1;
  padding: 12px 24px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 1rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
}

.tab-btn:hover {
  color: var(--text-primary);
}

.tab-btn.active {
  color: #00d4aa;
  background: rgba(0, 212, 170, 0.1);
}

.tab-btn.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: #00d4aa;
}
</style>
