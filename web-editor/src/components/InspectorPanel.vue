<template>
  <div class="inspector-panel">
    <div class="panel-header">
      <h3>Inspector</h3>
      <span class="entity-type">{{ entity.type }}</span>
    </div>

    <div class="properties">
      <!-- Transform Component -->
      <div class="component">
        <div class="component-header">
          <span class="component-icon">📐</span>
          <span class="component-name">Transform</span>
        </div>

        <div class="inspector-property">
          <label class="property-label">Position</label>
          <div class="vec3-input">
            <input type="number" step="0.1" v-model.number="entity.transform.position.x" placeholder="X">
            <input type="number" step="0.1" v-model.number="entity.transform.position.y" placeholder="Y">
            <input type="number" step="0.1" v-model.number="entity.transform.position.z" placeholder="Z">
          </div>
        </div>

        <div class="inspector-property">
          <label class="property-label">Rotation</label>
          <div class="vec3-input">
            <input type="number" step="1" v-model.number="entity.transform.rotation.x" placeholder="X">
            <input type="number" step="1" v-model.number="entity.transform.rotation.y" placeholder="Y">
            <input type="number" step="1" v-model.number="entity.transform.rotation.z" placeholder="Z">
          </div>
        </div>

        <div class="inspector-property">
          <label class="property-label">Scale</label>
          <div class="vec3-input">
            <input type="number" step="0.1" v-model.number="entity.transform.scale.x" placeholder="X">
            <input type="number" step="0.1" v-model.number="entity.transform.scale.y" placeholder="Y">
            <input type="number" step="0.1" v-model.number="entity.transform.scale.z" placeholder="Z">
          </div>
        </div>
      </div>

      <!-- Mesh Renderer Component (if applicable) -->
      <div v-if="entity.type === 'mesh'" class="component">
        <div class="component-header">
          <span class="component-icon">🎨</span>
          <span class="component-name">Mesh Renderer</span>
        </div>

        <div class="inspector-property">
          <label class="property-label">Mesh</label>
          <select class="property-input" v-model="entity.meshRenderer.mesh">
            <option value="cube">Cube</option>
            <option value="sphere">Sphere</option>
            <option value="plane">Plane</option>
            <option value="cylinder">Cylinder</option>
          </select>
        </div>

        <div class="inspector-property">
          <label class="property-label">Material</label>
          <select class="property-input" v-model="entity.meshRenderer.material">
            <option value="default">Default</option>
            <option value="metal">Metal</option>
            <option value="wood">Wood</option>
            <option value="plastic">Plastic</option>
          </select>
        </div>

        <div class="inspector-property">
          <label class="property-label">Color</label>
          <input type="color" class="property-input" v-model="entity.meshRenderer.color">
        </div>
      </div>

      <!-- Light Component (if applicable) -->
      <div v-if="entity.type === 'light'" class="component">
        <div class="component-header">
          <span class="component-icon">💡</span>
          <span class="component-name">Light</span>
        </div>

        <div class="inspector-property">
          <label class="property-label">Type</label>
          <select class="property-input" v-model="entity.light.type">
            <option value="directional">Directional</option>
            <option value="point">Point</option>
            <option value="spot">Spot</option>
          </select>
        </div>

        <div class="inspector-property">
          <label class="property-label">Color</label>
          <input type="color" class="property-input" v-model="entity.light.color">
        </div>

        <div class="inspector-property">
          <label class="property-label">Intensity</label>
          <input type="range" min="0" max="10" step="0.1" v-model.number="entity.light.intensity">
          <span class="property-value">{{ entity.light.intensity }}</span>
        </div>

        <div v-if="entity.light.type === 'point' || entity.light.type === 'spot'" class="inspector-property">
          <label class="property-label">Range</label>
          <input type="range" min="1" max="100" step="1" v-model.number="entity.light.range">
          <span class="property-value">{{ entity.light.range }}</span>
        </div>
      </div>

      <!-- Camera Component (if applicable) -->
      <div v-if="entity.type === 'camera'" class="component">
        <div class="component-header">
          <span class="component-icon">📹</span>
          <span class="component-name">Camera</span>
        </div>

        <div class="inspector-property">
          <label class="property-label">FOV</label>
          <input type="range" min="10" max="120" step="1" v-model.number="entity.camera.fov">
          <span class="property-value">{{ entity.camera.fov }}°</span>
        </div>

        <div class="inspector-property">
          <label class="property-label">Near Clip</label>
          <input type="number" step="0.1" class="property-input" v-model.number="entity.camera.near">
        </div>

        <div class="inspector-property">
          <label class="property-label">Far Clip</label>
          <input type="number" step="1" class="property-input" v-model.number="entity.camera.far">
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'InspectorPanel',
  props: {
    entity: {
      type: Object,
      required: true
    }
  }
}
</script>

<style scoped>
.inspector-panel {
  position: absolute;
  right: 0;
  top: 0;
  width: 320px;
  height: 100vh;
  background: #2a2a2a;
  border-left: 1px solid #444;
  overflow-y: auto;
}

.panel-header {
  padding: 16px;
  border-bottom: 1px solid #444;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.panel-header h3 {
  margin: 0;
  font-size: 14px;
}

.entity-type {
  font-size: 12px;
  color: #888;
  text-transform: uppercase;
}

.properties {
  padding: 16px;
}

.component {
  margin-bottom: 20px;
  border: 1px solid #444;
  border-radius: 4px;
  overflow: hidden;
}

.component-header {
  background: #333;
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: bold;
}

.component-icon {
  font-size: 14px;
}

.inspector-property {
  padding: 12px;
  border-bottom: 1px solid #333;
}

.inspector-property:last-child {
  border-bottom: none;
}

.property-label {
  display: block;
  margin-bottom: 6px;
  font-size: 11px;
  color: #ccc;
  text-transform: uppercase;
  font-weight: bold;
}

.property-input {
  width: 100%;
  padding: 6px 8px;
  background: #444;
  border: 1px solid #555;
  border-radius: 3px;
  color: #fff;
  font-size: 13px;
}

.property-input:focus {
  outline: none;
  border-color: #007acc;
}

.vec3-input {
  display: flex;
  gap: 4px;
}

.vec3-input input {
  flex: 1;
}

.property-value {
  margin-left: 8px;
  font-size: 12px;
  color: #aaa;
}
</style>