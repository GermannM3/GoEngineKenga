<template>
  <div class="hierarchy-panel">
    <div class="panel-header">
      <h3>Scene Hierarchy</h3>
      <button class="btn small" @click="createEntity">+ Create</button>
    </div>

    <div class="hierarchy-tree">
      <div
        v-for="entity in entities"
        :key="entity.id"
        class="hierarchy-item"
        :class="{ selected: entity.id === selectedEntityId }"
        @click="selectEntity(entity.id)"
      >
        <span class="entity-icon">{{ getEntityIcon(entity.type) }}</span>
        <span class="entity-name">{{ entity.name }}</span>
        <span class="entity-visibility" @click.stop="toggleVisibility(entity.id)">
          {{ entity.visible ? '👁' : '👁‍🗨' }}
        </span>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'HierarchyPanel',
  data() {
    return {
      entities: [
        { id: 'camera', name: 'Main Camera', type: 'camera', visible: true },
        { id: 'cube', name: 'Cube', type: 'mesh', visible: true },
        { id: 'sphere', name: 'Sphere', type: 'mesh', visible: true },
        { id: 'light', name: 'Directional Light', type: 'light', visible: true },
        { id: 'ground', name: 'Ground', type: 'mesh', visible: true }
      ],
      selectedEntityId: null
    }
  },
  methods: {
    getEntityIcon(type) {
      const icons = {
        camera: '📹',
        mesh: '📦',
        light: '💡',
        audio: '🔊'
      }
      return icons[type] || '📄'
    },

    selectEntity(id) {
      this.selectedEntityId = id
      this.$emit('entity-selected', id)
    },

    toggleVisibility(id) {
      const entity = this.entities.find(e => e.id === id)
      if (entity) {
        entity.visible = !entity.visible
      }
    },

    createEntity() {
      const newEntity = {
        id: 'entity_' + Date.now(),
        name: 'New Entity',
        type: 'mesh',
        visible: true
      }
      this.entities.push(newEntity)
      this.selectEntity(newEntity.id)
    }
  }
}
</script>

<style scoped>
.hierarchy-panel {
  height: 100%;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 12px;
  border-bottom: 1px solid #444;
  margin-bottom: 12px;
}

.panel-header h3 {
  margin: 0;
  font-size: 14px;
  color: #fff;
}

.btn.small {
  padding: 4px 8px;
  font-size: 11px;
}

.hierarchy-tree {
  overflow-y: auto;
}

.hierarchy-item {
  display: flex;
  align-items: center;
  padding: 6px 8px;
  margin: 1px 0;
  border-radius: 3px;
  cursor: pointer;
  user-select: none;
}

.hierarchy-item:hover {
  background: #3a3a3a;
}

.hierarchy-item.selected {
  background: #007acc;
}

.entity-icon {
  margin-right: 8px;
  font-size: 12px;
}

.entity-name {
  flex: 1;
  font-size: 13px;
}

.entity-visibility {
  margin-left: 8px;
  cursor: pointer;
  opacity: 0.7;
}

.entity-visibility:hover {
  opacity: 1;
}
</style>