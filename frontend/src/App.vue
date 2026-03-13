<template>
  <div class="container">
    <header class="header">
      <h1>🎬 FFmpeg 视频工具</h1>
      <p>PPT 视频压缩与 MOV 转 WebM</p>
    </header>

    <!-- FFmpeg 初始化状态 -->
    <div class="card" v-if="!ffmpegReady">
      <div class="ffmpeg-status">
        <div v-if="ffmpegError" class="status error">
          <span>❌ FFmpeg 初始化失败: {{ ffmpegError }}</span>
        </div>
        <div v-else class="ffmpeg-loading">
          <span class="loader"></span>
          <div class="ffmpeg-info">
            <p class="ffmpeg-status-text">{{ ffmpegStatus || '正在检查 FFmpeg...' }}</p>
            <div class="progress-bar" v-if="ffmpegProgress > 0">
              <div class="progress-fill" :style="{ width: ffmpegProgress + '%' }"></div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else>
      <!-- 功能切换 Tab -->
      <div class="card tab-bar">
        <button
          class="tab-btn"
          :class="{ active: activeTab === 'mov2webm' }"
          @click="activeTab = 'mov2webm'"
        >
          🔄 MOV 转 WebM
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeTab === 'ppt' }"
          @click="activeTab = 'ppt'"
        >
          📊 PPT 视频压缩
        </button>
      </div>

      <!-- MOV to WebM 功能 -->
      <template v-if="activeTab === 'mov2webm'">
        <div class="card">
          <!-- Initial State: Drop Zone -->
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

      <!-- File List State -->
      <div v-else-if="files.length > 0 && state !== 'complete'">
        <!-- File List Header -->
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

        <!-- File List -->
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
                <!-- Per-file progress bar -->
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

        <!-- Quality Settings -->
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

        <!-- Convert Button -->
        <div class="button-group">
          <button 
            class="btn btn-primary btn-block"
            @click="startBatchConversion"
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

        <!-- Overall Progress -->
        <div class="progress-container" v-if="state === 'converting'">
          <div class="progress-header">
            <span class="progress-label">总体进度</span>
            <span class="progress-percent">{{ completedCount }} / {{ files.length }}</span>
          </div>
          <div class="progress-bar">
            <div class="progress-fill overall" :style="{ width: overallProgress + '%' }"></div>
          </div>
        </div>

        <!-- Global Error Message -->
        <div class="status error" v-if="errorMessage" style="margin-top: 16px;">
          ⚠️ {{ errorMessage }}
        </div>
      </div>

      <!-- Complete State -->
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
        <!-- End tag for <div class="card"> -> -->
        </div>

        <!-- Tips -->
        <div class="card" style="padding: 20px;">
          <p style="color: var(--text-secondary); font-size: 0.9rem; text-align: center;">
            💡 提示: 输出文件将保存在源文件同目录下，文件名后缀为 <code>.webm</code>
          </p>
        </div>
      </template>

      <!-- PPT 压缩功能 -->
      <template v-if="activeTab === 'ppt'">
        <div class="card">
          <!-- PPT 文件选择 -->
          <div v-if="pptFiles.length === 0 && pptState === 'idle'" class="drop-zone" @click="selectPPTFiles">
            <span class="drop-zone-icon">📊</span>
            <p class="drop-zone-text">点击选择 PPT 文件</p>
            <p class="drop-zone-hint">支持 .pptx 格式，支持多选，自动识别母版视频</p>
          </div>

          <!-- PPT 列表状态 -->
          <div v-else-if="pptFiles.length > 0 && pptState !== 'complete'">
            <div class="file-list-header">
              <span class="file-count">
                <span class="file-count-icon">📂</span>
                已选择 {{ pptFiles.length }} 个 PPT 文件
              </span>
              <div class="flex gap-2">
                <button class="btn-text" @click="selectPPTFiles" :disabled="pptState !== 'idle'">
                  <span>➕</span> 添加
                </button>
                <button class="btn-text danger" @click="resetPPT" :disabled="pptState !== 'idle'">
                  <span>🗑️</span> 清空
                </button>
              </div>
            </div>

            <div class="file-list mb-4">
              <div 
                v-for="(ppt, index) in pptFiles" 
                :key="ppt.path"
                class="file-item"
                :class="{ 
                  'converting': pptCurrentFileIndex === index && pptState !== 'idle',
                  'completed': ppt.status === 'completed',
                  'error': ppt.status === 'error'
                }"
              >
                <div class="file-item-content">
                  <span class="file-status-icon">
                    <template v-if="ppt.status === 'completed'">✅</template>
                    <template v-else-if="ppt.status === 'error'">❌</template>
                    <template v-else-if="pptCurrentFileIndex === index && pptState !== 'idle'">
                      <span class="file-loader"></span>
                    </template>
                    <template v-else>📊</template>
                  </span>
                  <div class="file-item-details">
                    <div class="file-item-name">{{ getFileName(ppt.path) }}</div>
                    <div class="file-item-meta" v-if="ppt.status === 'completed'">
                      已压缩 · 节省 {{ formatFileSize(ppt.saved) }}
                    </div>
                    <div class="file-item-error" v-if="ppt.errorMessage">
                      {{ ppt.errorMessage }}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 质量与筛选设置 -->
            <div class="settings" v-if="pptState === 'idle'">
              <div class="flex items-center justify-between mb-4 px-2">
                <label class="flex items-center cursor-pointer select-none">
                  <input type="checkbox" v-model="pptOnlyMaster" class="mr-2 w-4 h-4">
                  <span class="text-sm font-medium">仅压缩母版中的视频</span>
                </label>
                <span class="text-xs text-gray-400">💡 母版视频通常出现在所有幻灯片背景中</span>
              </div>

              <div class="settings-row">
                <div>
                  <span class="settings-label">压缩质量 (CRF)</span>
                  <div class="settings-hint">推荐: 25-30</div>
                </div>
                <div class="range-wrapper">
                  <span class="range-label high">高质量</span>
                  <div class="range-container">
                    <input
                      type="range"
                      class="range-slider"
                      v-model.number="pptQuality"
                      :min="0"
                      :max="51"
                      :disabled="pptState !== 'idle'"
                    />
                  </div>
                  <span class="range-label low">低质量</span>
                </div>
                <span class="range-value" :class="pptQualityClass">{{ pptQuality }}</span>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="button-group mt-4">
              <button
                v-if="pptState === 'idle'"
                class="btn btn-primary btn-block"
                @click="startBatchPPTCompression"
                :disabled="pptFiles.length === 0"
              >
                <span>🚀</span>
                开始批量处理 ({{ pptFiles.length }} 个文件)
              </button>
              <button 
                v-else
                class="btn btn-danger btn-block"
                @click="cancelPPTCompression"
              >
                <span class="loader"></span>
                取消压缩
              </button>
            </div>

            <!-- 总体进度 -->
            <div class="progress-container mt-4" v-if="pptState !== 'idle'">
              <div class="progress-header">
                <span class="progress-label">
                  {{ pptState === 'extracting' ? '正在提取...' : (pptState === 'repackaging' ? '正在打包...' : '正在压缩...') }}
                </span>
                <span class="progress-percent">{{ Math.round(pptOverallProgress) }}%</span>
              </div>
              <div class="progress-bar">
                <div class="progress-fill overall" :style="{ width: pptOverallProgress + '%' }"></div>
              </div>
            </div>
          </div>

          <!-- PPT 完成状态 -->
          <div v-else-if="pptState === 'complete'" class="result">
            <span class="result-icon success">✅</span>
            <h2 class="result-title">PPT 批量压缩完成!</h2>
            <div class="result-summary">
              <p class="result-stats">
                已处理: {{ pptFiles.length }} 个文件<br>
                总节省空间: {{ formatFileSize(pptTotalSaved) }}
              </p>
            </div>
            <div class="result-actions">
              <button class="btn btn-primary" @click="resetPPT">
                🔄 继续压缩
              </button>
            </div>
          </div>

          <!-- 错误状态 -->
          <div v-if="pptErrorMessage" class="status error" style="margin-top: 16px;">
            ⚠️ {{ pptErrorMessage }}
          </div>
        </div>

        <!-- Tips -->
        <div class="card" style="padding: 20px;">
          <p style="color: var(--text-secondary); font-size: 0.9rem; text-align: center;">
            💡 提示: 压缩后的 PPT 文件会保存在原文件同目录下，文件名后缀为 <code>_compressed.pptx</code>
          </p>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue'

export default defineComponent({
  name: 'App',
  data() {
    return {
      // Tab 切换
      activeTab: 'mov2webm', // 'mov2webm' 或 'ppt'

      // MOV 转 WebM 数据
      files: [] as any[], // { path, info, status, outputPath, errorMessage }
      outputFolder: '',
      quality: 25,
      state: 'idle' as 'idle' | 'converting' | 'complete', // idle, converting, complete
      currentProgress: 0,
      currentFileIndex: -1,
      isDragOver: false,
      errorMessage: '',

      // PPT 压缩数据
      pptFiles: [] as any[], // { path, status, saved, errorMessage }
      pptOnlyMaster: true,
      pptQuality: 30,
      pptState: 'idle' as 'idle' | 'extracting' | 'compressing' | 'repackaging' | 'complete',
      pptCurrentFileIndex: -1,
      pptOverallProgress: 0,
      pptTotalSaved: 0,
      pptErrorMessage: '',

      // FFmpeg 状态
      ffmpegReady: false,
      ffmpegStatus: '',
      ffmpegProgress: 0,
      ffmpegError: ''
    }
  },
  computed: {
    qualityClass() {
      if (this.quality <= 20) return 'quality-high'
      if (this.quality <= 35) return 'quality-medium'
      return 'quality-low'
    },
    pptQualityClass() {
      if (this.pptQuality <= 20) return 'quality-high'
      if (this.pptQuality <= 35) return 'quality-medium'
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
    // Listen for events from Go backend
    const runtime = (window as any).runtime
    if (runtime) {
      // FFmpeg 状态事件
      runtime.EventsOn('ffmpeg:progress', (data: any) => {
        this.ffmpegStatus = data.status
        this.ffmpegProgress = data.progress
      })
      runtime.EventsOn('ffmpeg:ready', () => {
        this.ffmpegReady = true
        this.ffmpegStatus = 'FFmpeg 已就绪'
      })
      runtime.EventsOn('ffmpeg:error', (error: any) => {
        this.ffmpegError = error
      })

      // 转换状态事件
      runtime.EventsOn('conversion:progress', (progress: any) => {
        this.currentProgress = progress
      })
      runtime.EventsOn('conversion:complete', (path: any) => {
        if (this.currentFileIndex >= 0 && this.currentFileIndex < this.files.length) {
          this.files[this.currentFileIndex].status = 'completed'
          this.files[this.currentFileIndex].outputPath = path
        }
      })
      runtime.EventsOn('conversion:error', (error: any) => {
        if (this.currentFileIndex >= 0 && this.currentFileIndex < this.files.length) {
          this.files[this.currentFileIndex].status = 'error'
          this.files[this.currentFileIndex].errorMessage = error
        }
      })
      runtime.EventsOn('conversion:cancelled', () => {
        this.errorMessage = '转换已取消'
        this.state = 'idle'
        this.currentProgress = 0
        this.currentFileIndex = -1
      })

      // PPT 压缩事件
      runtime.EventsOn('ppt:progress', (data: any) => {
        this.pptOverallProgress = data.overallProgress
      })
      runtime.EventsOn('ppt:status', (status: any) => {
        this.pptState = status
      })
      runtime.EventsOn('ppt:cancelled', () => {
        this.pptErrorMessage = '压缩已取消'
        this.pptState = 'idle'
      })
    }

    // 主动检查 FFmpeg 状态，防止错过后端事件
    this.checkFFmpegStatus()
  },
  methods: {
    async checkFFmpegStatus() {
      try {
        const status = await (window as any).go.main.App.CheckFFmpegStatus()
        if (status && status.installed) {
          this.ffmpegReady = true
          this.ffmpegStatus = 'FFmpeg 已就绪'
        }
      } catch (error) {
        console.error('Failed to check FFmpeg status:', error)
      }
    },

    async selectFiles() {
      try {
        const selectedFiles = await (window as any).go.main.App.SelectInputFiles()
        if (selectedFiles && selectedFiles.length > 0) {
          this.errorMessage = ''
          // 添加新文件，避免重复
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
              // 异步加载视频信息
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
      // Note: Wails handles file drops differently, using the select dialog
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
    
    async startBatchConversion() {
      if (this.files.length === 0) {
        this.errorMessage = '请先选择文件'
        return
      }
      
      this.state = 'converting'
      this.currentProgress = 0
      this.currentFileIndex = 0
      this.errorMessage = ''
      
      // 重置所有文件状态
      for (const file of this.files) {
        file.status = 'pending'
        file.errorMessage = ''
      }
      
      // 依次转换每个文件
      for (let i = 0; i < this.files.length; i++) {
        if (this.state !== 'converting') break // 被取消
        
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
      } catch (error) {
        console.error('Failed to cancel conversion:', error)
      }
    },
    
    async openOutputFolder() {
      // 打开第一个成功转换的文件所在目录
      const successFile = this.files.find(f => f.status === 'completed')
      if (successFile && successFile.outputPath) {
        try {
          await (window as any).go.main.App.OpenFileInExplorer(successFile.outputPath)
        } catch (error: any) {
          console.error('Failed to open explorer:', error)
        }
      }
    },

    async openFileInExplorer(path: string) {
      if (path) {
        try {
          await (window as any).go.main.App.OpenFileInExplorer(path)
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
    },

    // ========== PPT 压缩方法 ==========
    async selectPPTFiles() {
      try {
        const selectedFiles = await (window as any).go.main.App.SelectPPTFiles()
        if (selectedFiles && selectedFiles.length > 0) {
          this.pptErrorMessage = ''
          for (const filePath of selectedFiles) {
            if (!this.pptFiles.find(f => f.path === filePath)) {
              this.pptFiles.push({
                path: filePath,
                status: 'pending',
                saved: 0,
                errorMessage: ''
              })
            }
          }
        }
      } catch (error: any) {
        this.pptErrorMessage = '选择文件失败: ' + error.message
      }
    },

    async startBatchPPTCompression() {
      if (this.pptFiles.length === 0) {
        this.pptErrorMessage = '请先选择 PPT 文件'
        return
      }

      this.pptState = 'extracting'
      this.pptErrorMessage = ''
      this.pptTotalSaved = 0

      for (let i = 0; i < this.pptFiles.length; i++) {
        if ((this.pptState as any) === 'idle') break // 取消

        this.pptCurrentFileIndex = i
        this.pptOverallProgress = (i / this.pptFiles.length) * 100

        try {
          const result = await (window as any).go.main.App.ProcessPPTFile(
            this.pptFiles[i].path,
            this.pptQuality,
            this.pptOnlyMaster
          )

          if (result.success) {
            this.pptFiles[i].status = 'completed'
            this.pptFiles[i].saved = result.totalSaved
            this.pptTotalSaved += result.totalSaved
          } else {
            this.pptFiles[i].status = 'error'
            this.pptFiles[i].errorMessage = result.message
          }
        } catch (error: any) {
          this.pptFiles[i].status = 'error'
          this.pptFiles[i].errorMessage = error.message
        }
      }

      if ((this.pptState as any) !== 'idle') {
        this.pptState = 'complete'
        this.pptCurrentFileIndex = -1
      }
    },

    async cancelPPTCompression() {
      try {
        await (window as any).go.main.App.CancelPPTCompression()
        this.pptState = 'idle'
        this.pptErrorMessage = '处理已取消'
      } catch (error: any) {
        console.error('Failed to cancel compression:', error)
      }
    },

    resetPPT() {
      this.pptFiles = []
      this.pptState = 'idle'
      this.pptCurrentFileIndex = -1
      this.pptOverallProgress = 0
      this.pptTotalSaved = 0
      this.pptErrorMessage = ''
    },

    getFileNameWithoutExt(filePath: any) {
      if (!filePath) return ''
      const name = filePath.split(/[/\\]/).pop()
      return name.replace(/\.[^/.]+$/, '')
    },

    formatFileSize(bytes: any) {
      if (!bytes) return '0 B'
      const units = ['B', 'KB', 'MB', 'GB', 'TB']
      let size = bytes
      let unitIndex = 0
      while (size >= 1024 && unitIndex < units.length - 1) {
        size /= 1024
        unitIndex++
      }
      return size.toFixed(2) + ' ' + units[unitIndex]
    }
  }
})
</script>

<style scoped>
/* Tab Bar Styles */
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
