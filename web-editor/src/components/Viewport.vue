<template>
  <div class="viewport" ref="viewport"></div>
</template>

<script>
import * as THREE from 'three'

export default {
  name: 'Viewport',
  mounted() {
    this.initThreeJS()
    this.animate()
  },
  beforeUnmount() {
    this.cleanup()
  },
  methods: {
    initThreeJS() {
      // Scene
      this.scene = new THREE.Scene()
      this.scene.background = new THREE.Color(0x1a1a1a)

      // Camera
      this.camera = new THREE.PerspectiveCamera(
        75,
        this.$refs.viewport.clientWidth / this.$refs.viewport.clientHeight,
        0.1,
        1000
      )
      this.camera.position.set(5, 5, 5)
      this.camera.lookAt(0, 0, 0)

      // Renderer
      this.renderer = new THREE.WebGLRenderer({ antialias: true })
      this.renderer.setSize(
        this.$refs.viewport.clientWidth,
        this.$refs.viewport.clientHeight
      )
      this.renderer.shadowMap.enabled = true
      this.renderer.shadowMap.type = THREE.PCFSoftShadowMap
      this.$refs.viewport.appendChild(this.renderer.domElement)

      // Controls
      this.controls = new THREE.OrbitControls(this.camera, this.renderer.domElement)

      // Lighting
      this.setupLighting()

      // Grid helper
      const gridHelper = new THREE.GridHelper(20, 20)
      this.scene.add(gridHelper)

      // Test objects
      this.addTestObjects()

      // Handle resize
      window.addEventListener('resize', this.onWindowResize)
    },

    setupLighting() {
      // Ambient light
      const ambientLight = new THREE.AmbientLight(0x404040, 0.4)
      this.scene.add(ambientLight)

      // Directional light
      const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8)
      directionalLight.position.set(10, 10, 5)
      directionalLight.castShadow = true
      directionalLight.shadow.mapSize.width = 2048
      directionalLight.shadow.mapSize.height = 2048
      this.scene.add(directionalLight)
    },

    addTestObjects() {
      // Cube
      const cubeGeometry = new THREE.BoxGeometry(1, 1, 1)
      const cubeMaterial = new THREE.MeshLambertMaterial({ color: 0x00ff00 })
      const cube = new THREE.Mesh(cubeGeometry, cubeMaterial)
      cube.position.set(0, 0.5, 0)
      cube.castShadow = true
      cube.receiveShadow = true
      this.scene.add(cube)

      // Sphere
      const sphereGeometry = new THREE.SphereGeometry(0.5, 32, 32)
      const sphereMaterial = new THREE.MeshLambertMaterial({ color: 0xff0000 })
      const sphere = new THREE.Mesh(sphereGeometry, sphereMaterial)
      sphere.position.set(2, 0.5, 0)
      sphere.castShadow = true
      sphere.receiveShadow = true
      this.scene.add(sphere)

      // Ground plane
      const planeGeometry = new THREE.PlaneGeometry(20, 20)
      const planeMaterial = new THREE.MeshLambertMaterial({ color: 0x808080 })
      const plane = new THREE.Mesh(planeGeometry, planeMaterial)
      plane.rotation.x = -Math.PI / 2
      plane.receiveShadow = true
      this.scene.add(plane)
    },

    animate() {
      requestAnimationFrame(this.animate)

      // Update controls
      if (this.controls) {
        this.controls.update()
      }

      // Render scene
      this.renderer.render(this.scene, this.camera)
    },

    onWindowResize() {
      this.camera.aspect = this.$refs.viewport.clientWidth / this.$refs.viewport.clientHeight
      this.camera.updateProjectionMatrix()
      this.renderer.setSize(
        this.$refs.viewport.clientWidth,
        this.$refs.viewport.clientHeight
      )
    },

    play() {
      // Start scene playback
      console.log('Playing scene...')
    },

    pause() {
      // Pause scene playback
      console.log('Pausing scene...')
    },

    stop() {
      // Stop scene playback
      console.log('Stopping scene...')
    },

    cleanup() {
      window.removeEventListener('resize', this.onWindowResize)
      if (this.renderer) {
        this.renderer.dispose()
      }
    }
  }
}
</script>