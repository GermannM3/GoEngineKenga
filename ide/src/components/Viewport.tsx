export function Viewport() {
  return (
    <div className="viewport">
      <div className="viewport-toolbar">
        <span className="viewport-title">Scene</span>
        <div className="viewport-controls">
          <button title="Perspective">Persp</button>
          <button title="Wireframe">Wire</button>
          <button title="Shaded">Shaded</button>
          <button title="Grid">Grid</button>
        </div>
      </div>
      <div className="viewport-canvas">
        <div className="viewport-placeholder">
          <span>3D Viewport</span>
          <small>Connect to GoEngineKenga via WebSocket</small>
        </div>
      </div>
    </div>
  );
}
