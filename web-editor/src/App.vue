<template>
  <div class="editor-layout">
    <!-- Sidebar -->
    <div class="sidebar">
      <!-- Panel Tabs -->
      <div class="panel-tabs">
        <button
          class="panel-tab"
          :class="{ active: activePanel === 'hierarchy' }"
          @click="activePanel = 'hierarchy'"
        >
          Hierarchy
        </button>
        <button
          class="panel-tab"
          :class="{ active: activePanel === 'project' }"
          @click="activePanel = 'project'"
        >
          Project
        </button>
      </div>

      <!-- Panel Content -->
      <div class="panel-content">
        <HierarchyPanel v-if="activePanel === 'hierarchy'" />
        <ProjectPanel v-if="activePanel === 'project'" />
      </div>
    </div>

    <!-- Main Content -->
    <div class="main-content">
      <!-- Toolbar -->
      <div class="toolbar">
        <button class="btn" @click="playScene">▶ Play</button>
        <button class="btn secondary" @click="pauseScene">⏸ Pause</button>
        <button class="btn secondary" @click="stopScene">⏹ Stop</button>
        <span style="margin-left: auto; color: #aaa;">GoEngineKenga Editor v0.1</span>
      </div>

      <!-- Viewport -->
      <div class="viewport-container">
        <Viewport ref="viewport" />
      </div>

      <!-- Status Bar -->
      <div class="status-bar">
        <span>Scene: {{ currentScene }}</span>
        <span style="margin-left: auto;">Entities: {{ entitiesCount }}</span>
        <span style="margin-left: 16px;">FPS: {{ fps }}</span>
      </div>
    </div>

    <!-- Inspector (floating panel) -->
    <InspectorPanel v-if="selectedEntity" :entity="selectedEntity" />
  </div>
</template>

<script>
import HierarchyPanel from './components/HierarchyPanel.vue'
import ProjectPanel from './components/ProjectPanel.vue'
import Viewport from './components/Viewport.vue'
import InspectorPanel from './components/InspectorPanel.vue'

export default {
  name: 'App',
  components: {
    HierarchyPanel,
    ProjectPanel,
    Viewport,
    InspectorPanel
  },
  data() {
    return {
      activePanel: 'hierarchy',
      currentScene: 'Main Scene',
      entitiesCount: 0,
      fps: 60,
      selectedEntity: null
    }
  },
  methods: {
    playScene() {
      this.$refs.viewport.play()
    },
    pauseScene() {
      this.$refs.viewport.pause()
    },
    stopScene() {
      this.$refs.viewport.stop()
    }
  }
}
</script>