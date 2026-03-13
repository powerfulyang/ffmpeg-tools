<template>
  <div class="card">
    <!-- 拖放区域 -->
    <div
      v-if="files.length === 0 && state === 'idle'"
      class="drop-zone"
      :class="{ 'drag-over': isDragOver }"
      @click="selectFiles"
      @dragover.prevent="isDragOver = true"
      @dragleave="isDragOver = false"
      @drop.prevent="handleDrop"
    >
      <span class="drop-zone-icon">📁</span>
      <p class="drop-zone-text">点击选择或拖放 MOV 文件</p>
      <p class="drop-zone-hint">支持选择多个带透明通道的 MOV 视频文件</p>
    </div>

    <!-- 文件列表 -->
    <div v-else-if="files.length > 0 && state !== 'complete'">
      <!-- 文件列表头部 -->
      <div class="file-list-header">
        <span class="file-count">
          <span class="file-count-icon">📂</span>
          已选择 {{ files.length }} 个文件
        </span>
        <div class="file-list-actions">
          <button class="btn-text" @click="selectFiles" :disabled="state === 'converting'">
            <span>➕</span> 添加
          </button>
          <button class="btn-text danger" @click="clearAllFiles" :disabled="state === 'converting'">
            <span>🗑️</span> 清空
          </button>
        </div>
      </div>

      <!-- 文件列表 -->
      <div class="file-list">
        <div
          class="file-item"
          v-for="(file, index) in files"
          :key="file.path"
          :class="{
            'converting': currentFileIndex === index && state === 'converting',
            'completed': file.status === 'completed',
            'error': file.status === 'error'
          }"
        >
          <div class="file-item-content">
            <span class="file-status-icon">
              <template v-if="file.status === 'completed'">✅</template>
              <template v-else-if="file.status === 'error'">❌</template>
              <template v-else-if="currentFileIndex === index && state === 'converting'">
                <span class="file-loader"></span>
              </template>
              <template v-else>🎥</template>
            </span>
            <div class="file-item-details">
              <div class="file-item-name">{{ getFileName(file.path) }}</div>
              <div class="file-item-meta" v-if="file.info">
                {{ file.info.width }}×{{ file.info.height }} · {{ formatDuration(file.info.duration) }}
                <span :class="file.info.hasAlpha ? 'alpha-badge-small' : 'no-alpha-badge-small'">
                  {{ file.info.hasAlpha ? '透明' : '不透明' }}
                </span>
              </div>
              <div class="file-item-error" v-if="file.errorMessage">
                {{ file.errorMessage }}
              </div>
              <!-- 单个文件进度条 -->
              <div class="file-item-progress" v-if="currentFileIndex === index && state === 'converting'">
                <div class="progress-bar small">
                  <div class="progress-fill" :style="{ width: currentProgress + '%' }"></div>
                </div>
                <span class="progress-text">{{ Math.round(currentProgress) }}%</span>
              </div>
            </div>
            <button
              class="file-item-remove"
              @click="removeFile(index)"
              title="移除文件"
              :disabled="state === 'converting'"
            >✕</button>
          </div>
        </div>
      </div>

      <!-- 质量设置 -->
      <div class="settings">
        <div class="quality-warning">
          ⚠️ <strong>注意：数值越低质量越高</strong>，文件也越大
        </div>
        <div class="settings-row">
          <div>
            <span class="settings-label">输出质量 (CRF)</span>
            <div class="settings-hint">推荐范围: 20-30</div>
          </div>
          <div class="range-wrapper">
            <span class="range-label high">高质量</span>
            <div class="range-container">
              <input
                type="range"
                class="range-slider"
                v-model.number="quality"
                :min="0"
                :max="63"
                :disabled="state === 'converting'"
              />
            </div>
            <span class="range-label low">低质量</span>
          </div>
          <span class="range-value" :class="qualityClass">{{ quality }}</span>
        </div>
      </div>

      <!-- 转换按钮 -->
      <div class="button-group">
        <button
          class="btn btn-primary btn-block"
          @click="startConversion"
          :disabled="files.length === 0"
          v-if="state !== 'converting'"
        >
          <span>🚀</span>
          开始批量转换 ({{ files.length }} 个文件)
        </button>
        <button
          class="btn btn-danger btn-block"
          @click="cancelConversion"
          v-else
        >
          <span class="loader"></span>
          取消转换
        </button>
      </div>

      <!-- 总体进度 -->
      <div class="progress-container" v-if="state === 'converting'">
        <div class="progress-header">
          <span class="progress-label">总体进度</span>
          <span class="progress-percent">{{ completedCount }} / {{ files.length }}</span>
        </div>
        <div class="progress-bar">
          <div class="progress-fill overall" :style="{ width: overallProgress + '%' }"></div>
        </div>
      </div>

      <!-- 错误信息 -->
      <div class="status error" v-if="errorMessage" style="margin-top: 16px;">
        ⚠️ {{ errorMessage }}
      </div>
    </div>

    <!-- 完成状态 -->
    <div v-else-if="state === 'complete'" class="result">
      <span class="result-icon success">✅</span>
      <h2 class="result-title">批量转换完成!</h2>
      <div class="result-summary">
        <p class="result-stats">
          成功: <span class="success-count">{{ successCount }}</span> ·
          失败: <span class="error-count">{{ errorCount }}</span>
        </p>
      </div>
      <div class="result-actions">
        <button class="btn btn-success" @click="openOutputFolder" v-if="successCount > 0">
          📂 打开输出目录
        </button>
        <button class="btn btn-primary" @click="resetAll">
          🔄 继续转换
        </button>
      </div>
    </div>
  </div>

  <!-- 提示 -->
  <div class="card" style="padding: 20px;">
    <p style="color: var(--text-secondary); font-size: 0.9rem; text-align: center;">
      💡 提示: 输出文件将保存在源文件同目录下，文件名后缀为 <code>.webm</code>
    </p>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue'

export default defineComponent({
  name: 'Mov2Webm',
  data() {
    return {
      files: [] as any[],
      outputFolder: '',
      quality: 25,
      state: 'idle' as 'idle' | 'converting' | 'complete',
      currentProgress: 0,
      currentFileIndex: -1,
      isDragOver: false,
      errorMessage: ''
    }
  },
  computed: {
    qualityClass() {
      if (this.quality <= 20) return 'quality-high'
      if (this.quality <= 35) return 'quality-medium'
      return 'quality-low'
    },
    completedCount() {
      return this.files.filter(f => f.status === 'completed' || f.status === 'error').length
    },
    successCount() {
      return this.files.filter(f => f.status === 'completed').length
    },
    errorCount() {
      return this.files.filter(f => f.status === 'error').length
    },
    overallProgress() {
      if (this.files.length === 0) return 0
      const baseProgress = (this.completedCount / this.files.length) * 100
      const currentFileProgress = (this.currentProgress / 100) / this.files.length * 100
      return Math.min(baseProgress + currentFileProgress, 100)
    }
  },
  mounted() {
    // 监听转换事件
    if ((window as any).runtime) {
      (window as any).runtime.EventsOn('conversion:progress', (progress: any) => {
        this.currentProgress = progress
      })
      (window as any).runtime.EventsOn('conversion:complete', (path: any) => {
        if (this.currentFileIndex >= 0 && this.currentFileIndex < this.files.length) {
          this.files[this.currentFileIndex].status = 'completed'
          this.files[this.currentFileIndex].outputPath = path
        }
      })
      (window as any).runtime.EventsOn('conversion:error', (error: any) => {
        if (this.currentFileIndex >= 0 && this.currentFileIndex < this.files.length) {
          this.files[this.currentFileIndex].status = 'error'
          this.files[this.currentFileIndex].errorMessage = error
        }
      })
      (window as any).runtime.EventsOn('conversion:cancelled', () => {
        this.errorMessage = '转换已取消'
        this.state = 'idle'
        this.currentProgress = 0
        this.currentFileIndex = -1
      })
    }
  },
  methods: {
    async selectFiles() {
      try {
        const selectedFiles = await (window as any).go.main.App.SelectInputFiles()
        if (selectedFiles && selectedFiles.length > 0) {
          this.errorMessage = ''
          for (const filePath of selectedFiles) {
            if (!this.files.find(f => f.path === filePath)) {
              const fileObj = {
                path: filePath,
                info: null,
                status: 'pending',
                outputPath: '',
                errorMessage: ''
              }
              this.files.push(fileObj)
              this.loadVideoInfo(fileObj)
            }
          }
        }
      } catch (error: any) {
        this.errorMessage = '选择文件失败: ' + error.message
      }
    },

    handleDrop(_event: any) {
      this.isDragOver = false
      this.selectFiles()
    },

    async loadVideoInfo(fileObj: any) {
      try {
        const info = await (window as any).go.main.App.GetVideoInfo(fileObj.path)
        fileObj.info = info
      } catch (error: any) {
        console.error('Failed to load video info:', error)
      }
    },

    getFileName(filePath: any) {
      if (!filePath) return ''
      return filePath.split(/[/\\]/).pop()
    },

    removeFile(index: any) {
      if (this.state === 'converting') return
      this.files.splice(index, 1)
    },

    clearAllFiles() {
      if (this.state === 'converting') return
      this.files = []
      this.errorMessage = ''
    },

    async startConversion() {
      if (this.files.length === 0) {
        this.errorMessage = '请先选择文件'
        return
      }

      this.state = 'converting'
      this.currentProgress = 0
      this.currentFileIndex = 0
      this.errorMessage = ''

      for (const file of this.files) {
        file.status = 'pending'
        file.errorMessage = ''
      }

      for (let i = 0; i < this.files.length; i++) {
        if (this.state !== 'converting') break

        this.currentFileIndex = i
        this.currentProgress = 0

        try {
          const result = await (window as any).go.main.App.ConvertToWebM(
            this.files[i].path,
            this.outputFolder,
            this.quality
          )

          if (result.success) {
            this.files[i].status = 'completed'
            this.files[i].outputPath = result.outputPath
          } else {
            this.files[i].status = 'error'
            this.files[i].errorMessage = result.message
          }
        } catch (error: any) {
          this.files[i].status = 'error'
          this.files[i].errorMessage = '转换失败: ' + error.message
        }
      }

      if (this.state === 'converting') {
        this.state = 'complete'
        this.currentFileIndex = -1
      }
    },

    async cancelConversion() {
      try {
        await (window as any).go.main.App.CancelConversion()
        this.state = 'idle'
        this.currentFileIndex = -1
        this.currentProgress = 0
      } catch (error: any) {
        console.error('Failed to cancel conversion:', error)
      }
    },

    async openOutputFolder() {
      const successFile = this.files.find(f => f.status === 'completed')
      if (successFile && successFile.outputPath) {
        try {
          await (window as any).go.main.App.OpenFileInExplorer(successFile.outputPath)
        } catch (error: any) {
          console.error('Failed to open explorer:', error)
        }
      }
    },

    resetAll() {
      this.files = []
      this.state = 'idle'
      this.currentProgress = 0
      this.currentFileIndex = -1
      this.errorMessage = ''
    },

    formatDuration(seconds: any) {
      if (!seconds) return '--'
      const secs = parseFloat(seconds)
      const mins = Math.floor(secs / 60)
      const remainingSecs = Math.floor(secs % 60)
      return `${mins}:${remainingSecs.toString().padStart(2, '0')}`
    }
  }
})
</script>

<style scoped>
.drop-zone {
  padding: 48px 24px;
  text-align: center;
  border: 2px dashed var(--border-color);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.drop-zone:hover,
.drop-zone.drag-over {
  border-color: #00d4aa;
  background: rgba(0, 212, 170, 0.05);
}

.drop-zone-icon {
  font-size: 3rem;
  margin-bottom: 16px;
}

.drop-zone-text {
  color: var(--text-primary);
  font-size: 1.1rem;
  margin-bottom: 8px;
}

.drop-zone-hint {
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.file-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 0 4px;
}

.file-count {
  color: var(--text-primary);
  font-weight: 500;
}

.file-count-icon {
  margin-right: 8px;
}

.file-list-actions {
  display: flex;
  gap: 8px;
}

.btn-text {
  padding: 6px 12px;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-text:hover:not(:disabled) {
  border-color: #00d4aa;
  color: #00d4aa;
}

.btn-text.danger:hover:not(:disabled) {
  border-color: #ff4757;
  color: #ff4757;
}

.btn-text:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.file-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.file-item {
  padding: 12px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 8px;
  border: 1px solid var(--border-color);
  transition: all 0.2s ease;
}

.file-item.converting {
  border-color: #00d4aa;
  background: rgba(0, 212, 170, 0.05);
}

.file-item.completed {
  border-color: #2ed573;
  background: rgba(46, 213, 115, 0.05);
}

.file-item.error {
  border-color: #ff4757;
  background: rgba(255, 71, 87, 0.05);
}

.file-item-content {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.file-status-icon {
  font-size: 1.2rem;
  margin-top: 2px;
}

.file-loader {
  width: 16px;
  height: 16px;
  border: 2px solid var(--border-color);
  border-top-color: #00d4aa;
  border-radius: 50%;
  display: inline-block;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.file-item-details {
  flex: 1;
  min-width: 0;
}

.file-item-name {
  color: var(--text-primary);
  font-weight: 500;
  margin-bottom: 4px;
  word-break: break-all;
}

.file-item-meta {
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.alpha-badge-small,
.no-alpha-badge-small {
  display: inline-block;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.75rem;
  margin-left: 8px;
}

.alpha-badge-small {
  background: rgba(0, 212, 170, 0.15);
  color: #00d4aa;
}

.no-alpha-badge-small {
  background: rgba(255, 193, 7, 0.15);
  color: #ffc107;
}

.file-item-error {
  color: #ff4757;
  font-size: 0.8rem;
  margin-top: 4px;
}

.file-item-progress {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-item-remove {
  padding: 4px 8px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 1rem;
  cursor: pointer;
  transition: color 0.2s ease;
}

.file-item-remove:hover:not(:disabled) {
  color: #ff4757;
}

.file-item-remove:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.settings {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--border-color);
}

.quality-warning {
  color: #ffc107;
  font-size: 0.85rem;
  margin-bottom: 12px;
}

.settings-row {
  display: flex;
  align-items: center;
  gap: 16px;
}

.settings-label {
  color: var(--text-primary);
  font-weight: 500;
}

.settings-hint {
  color: var(--text-secondary);
  font-size: 0.8rem;
  margin-top: 2px;
}

.range-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.range-label {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.range-container {
  flex: 1;
}

.range-slider {
  width: 100%;
  height: 6px;
  -webkit-appearance: none;
  appearance: none;
  background: var(--border-color);
  border-radius: 3px;
  outline: none;
}

.range-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 18px;
  height: 18px;
  background: #00d4aa;
  border-radius: 50%;
  cursor: pointer;
  transition: transform 0.2s ease;
}

.range-slider::-webkit-slider-thumb:hover {
  transform: scale(1.1);
}

.range-value {
  min-width: 32px;
  padding: 4px 8px;
  border-radius: 6px;
  font-weight: 600;
  text-align: center;
}

.range-value.quality-high {
  background: rgba(46, 213, 115, 0.15);
  color: #2ed573;
}

.range-value.quality-medium {
  background: rgba(0, 212, 170, 0.15);
  color: #00d4aa;
}

.range-value.quality-low {
  background: rgba(255, 193, 7, 0.15);
  color: #ffc107;
}

.button-group {
  margin-top: 20px;
}

.btn {
  padding: 12px 24px;
  border: none;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-block {
  width: 100%;
  justify-content: center;
}

.btn-primary {
  background: linear-gradient(135deg, #00d4aa, #00b894);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 212, 170, 0.3);
}

.btn-danger {
  background: rgba(255, 71, 87, 0.1);
  color: #ff4757;
  border: 1px solid rgba(255, 71, 87, 0.3);
}

.btn-danger:hover:not(:disabled) {
  background: rgba(255, 71, 87, 0.2);
}

.btn-success {
  background: rgba(46, 213, 115, 0.1);
  color: #2ed573;
  border: 1px solid rgba(46, 213, 115, 0.3);
}

.btn-success:hover:not(:disabled) {
  background: rgba(46, 213, 115, 0.2);
}

.loader {
  width: 16px;
  height: 16px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  display: inline-block;
  animation: spin 1s linear infinite;
}

.progress-container {
  margin-top: 20px;
}

.progress-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.progress-label {
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.progress-percent {
  color: var(--text-primary);
  font-weight: 500;
}

.progress-bar {
  height: 8px;
  background: var(--border-color);
  border-radius: 4px;
  overflow: hidden;
}

.progress-bar.small {
  height: 4px;
  flex: 1;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #00d4aa, #00b894);
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 0.8rem;
  color: var(--text-secondary);
  min-width: 40px;
  text-align: right;
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

.result {
  text-align: center;
  padding: 24px;
}

.result-icon {
  font-size: 4rem;
  margin-bottom: 16px;
}

.result-title {
  color: var(--text-primary);
  margin-bottom: 16px;
}

.result-summary {
  margin-bottom: 24px;
}

.result-stats {
  color: var(--text-secondary);
}

.success-count {
  color: #2ed573;
  font-weight: 600;
}

.error-count {
  color: #ff4757;
  font-weight: 600;
}

.result-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}
</style>
