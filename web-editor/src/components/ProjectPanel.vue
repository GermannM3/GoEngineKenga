<template>
  <div class="project-panel">
    <div class="panel-header">
      <h3>Project</h3>
      <button class="btn small" @click="importAsset">📁 Import</button>
    </div>

    <div class="asset-tree">
      <div class="asset-folder" v-for="folder in assetFolders" :key="folder.name">
        <div class="folder-header" @click="toggleFolder(folder)">
          <span class="folder-icon">{{ folder.expanded ? '📂' : '📁' }}</span>
          <span class="folder-name">{{ folder.name }}</span>
        </div>

        <div v-if="folder.expanded" class="folder-contents">
          <div
            v-for="asset in folder.assets"
            :key="asset.name"
            class="asset-item"
            @click="selectAsset(asset)"
            :class="{ selected: asset.name === selectedAsset }"
          >
            <span class="asset-icon">{{ getAssetIcon(asset.type) }}</span>
            <span class="asset-name">{{ asset.name }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ProjectPanel',
  data() {
    return {
      selectedAsset: null,
      assetFolders: [
        {
          name: 'Scenes',
          expanded: true,
          assets: [
            { name: 'Main Scene.scene.json', type: 'scene' },
            { name: 'Menu Scene.scene.json', type: 'scene' }
          ]
        },
        {
          name: 'Models',
          expanded: true,
          assets: [
            { name: 'cube.glb', type: 'model' },
            { name: 'sphere.glb', type: 'model' },
            { name: 'ship.glb', type: 'model' }
          ]
        },
        {
          name: 'Textures',
          expanded: false,
          assets: [
            { name: 'wood.png', type: 'texture' },
            { name: 'metal.png', type: 'texture' },
            { name: 'grass.png', type: 'texture' }
          ]
        },
        {
          name: 'Audio',
          expanded: false,
          assets: [
            { name: 'ambient.wav', type: 'audio' },
            { name: 'explosion.wav', type: 'audio' },
            { name: 'music.mp3', type: 'audio' }
          ]
        },
        {
          name: 'Scripts',
          expanded: false,
          assets: [
            { name: 'PlayerController.go', type: 'script' },
            { name: 'GameManager.go', type: 'script' },
            { name: 'UIManager.go', type: 'script' }
          ]
        }
      ]
    }
  },
  methods: {
    toggleFolder(folder) {
      folder.expanded = !folder.expanded
    },

    getAssetIcon(type) {
      const icons = {
        scene: '🎬',
        model: '📦',
        texture: '🖼️',
        audio: '🔊',
        script: '📄'
      }
      return icons[type] || '📄'
    },

    selectAsset(asset) {
      this.selectedAsset = asset.name
      this.$emit('asset-selected', asset)
    },

    importAsset() {
      // Create file input
      const input = document.createElement('input')
      input.type = 'file'
      input.multiple = true
      input.accept = '.glb,.gltf,.png,.jpg,.wav,.mp3,.ogg,.go,.scene.json'

      input.onchange = (e) => {
        const files = Array.from(e.target.files)
        this.processImportedFiles(files)
      }

      input.click()
    },

    processImportedFiles(files) {
      files.forEach(file => {
        const type = this.getFileType(file.name)
        const folder = this.getFolderForType(type)

        if (folder) {
          folder.assets.push({
            name: file.name,
            type: type
          })
        }

        console.log('Imported:', file.name, 'as', type)
      })
    },

    getFileType(filename) {
      const ext = filename.split('.').pop().toLowerCase()
      const typeMap = {
        'glb': 'model',
        'gltf': 'model',
        'png': 'texture',
        'jpg': 'texture',
        'jpeg': 'texture',
        'wav': 'audio',
        'mp3': 'audio',
        'ogg': 'audio',
        'go': 'script',
        'json': 'scene'
      }
      return typeMap[ext] || 'unknown'
    },

    getFolderForType(type) {
      const folderMap = {
        'model': this.assetFolders.find(f => f.name === 'Models'),
        'texture': this.assetFolders.find(f => f.name === 'Textures'),
        'audio': this.assetFolders.find(f => f.name === 'Audio'),
        'script': this.assetFolders.find(f => f.name === 'Scripts'),
        'scene': this.assetFolders.find(f => f.name === 'Scenes')
      }
      return folderMap[type]
    }
  }
}
</script>

<style scoped>
.project-panel {
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
}

.btn.small {
  padding: 4px 8px;
  font-size: 11px;
}

.asset-tree {
  overflow-y: auto;
}

.asset-folder {
  margin-bottom: 8px;
}

.folder-header {
  display: flex;
  align-items: center;
  padding: 6px 8px;
  cursor: pointer;
  border-radius: 3px;
  user-select: none;
}

.folder-header:hover {
  background: #3a3a3a;
}

.folder-icon {
  margin-right: 8px;
  font-size: 14px;
}

.folder-name {
  font-size: 13px;
  font-weight: bold;
}

.folder-contents {
  margin-left: 20px;
  margin-top: 4px;
}

.asset-item {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  margin: 1px 0;
  border-radius: 3px;
  cursor: pointer;
  user-select: none;
}

.asset-item:hover {
  background: #3a3a3a;
}

.asset-item.selected {
  background: #007acc;
}

.asset-icon {
  margin-right: 8px;
  font-size: 12px;
}

.asset-name {
  font-size: 12px;
  flex: 1;
}
</style>