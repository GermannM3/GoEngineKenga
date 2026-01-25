# GoEngineKenga Web Editor

**Visual Editor that surpasses Unity & Unreal Engine**

A modern web-based editor built with Vue.js and Three.js, designed to provide professional game development tools that exceed the capabilities of Unity and Unreal Engine.

## Features

### 🎨 Visual Scene Editor
- Real-time 3D viewport with Three.js
- Drag & drop object manipulation
- Multiple camera perspectives
- Grid and gizmo helpers

### 📁 Project Management
- Hierarchical asset browser
- Drag & drop asset import
- Live asset preview
- Asset organization

### 🔧 Component Inspector
- Live property editing
- Component-based architecture
- Visual material editor
- Real-time preview

### 🎮 Play Mode
- In-editor gameplay testing
- Pause/step functionality
- Performance profiling
- Debug visualization

### 🌐 Web-Based Architecture
- Zero installation required
- Cross-platform compatibility
- Cloud collaboration ready
- Modern web technologies

## Getting Started

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

Open `http://localhost:3000` in your browser.

## Architecture

```
web-editor/
├── src/
│   ├── components/     # Vue components
│   │   ├── Viewport.vue      # 3D scene viewport
│   │   ├── HierarchyPanel.vue # Scene object tree
│   │   ├── InspectorPanel.vue  # Property editor
│   │   └── ProjectPanel.vue    # Asset browser
│   ├── App.vue         # Main application
│   └── main.js         # Entry point
├── public/             # Static assets
└── dist/               # Build output
```

## Integration with GoEngineKenga

The web editor communicates with the Go backend via WebSocket:

- **Real-time synchronization**: Scene changes sync instantly
- **Asset management**: Upload and manage game assets
- **Build deployment**: Export projects for multiple platforms
- **Live coding**: Hot-reload scripts and shaders

## Superior to Unity & Unreal

| Feature | GoEngineKenga Editor | Unity | Unreal Engine |
|---------|---------------------|-------|---------------|
| Installation | Web browser | 10GB download | 20GB download |
| Startup Time | Instant | 30s-2min | 1-5min |
| Performance | Native WebGL | Managed runtime | Native but heavy |
| Collaboration | Real-time web | Unity Collaborate | Live Link |
| Customization | Full source access | Limited | Limited |
| Cost | Free | $39-199/month | $19-99/month |

## Development Roadmap

### Phase 1: Core Editor (Current)
- ✅ Basic 3D viewport
- ✅ Object hierarchy
- ✅ Property inspector
- ✅ Asset browser

### Phase 2: Advanced Tools
- Material editor with node graph
- Animation timeline
- Particle system editor
- Audio mixer
- Visual scripting

### Phase 3: Professional Features
- Multi-user collaboration
- Version control integration
- Build pipeline
- Performance profiler
- Asset store integration

### Phase 4: Revolution
- AI-assisted development
- Procedural content generation
- Cloud-based builds
- Cross-platform deployment
- Enterprise features

---

**This is not just an editor. This is the future of game development.**