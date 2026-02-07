interface InspectorPanelProps {
  selectedEntity?: string | null;
}

export function InspectorPanel({ selectedEntity }: InspectorPanelProps) {
  return (
    <div className="inspector-panel">
      <div className="panel-header">
        <span>Inspector</span>
      </div>
      <div className="panel-content">
        {selectedEntity ? (
          <div className="inspector-props">
            <div className="prop-section">
              <div className="prop-section-title">Transform</div>
              <div className="prop-row">
                <label>Position</label>
                <span>0, 0, 0</span>
              </div>
              <div className="prop-row">
                <label>Rotation</label>
                <span>0°, 0°, 0°</span>
              </div>
              <div className="prop-row">
                <label>Scale</label>
                <span>1, 1, 1</span>
              </div>
            </div>
            <div className="prop-section">
              <div className="prop-section-title">Mesh</div>
              <div className="prop-row">
                <label>File</label>
                <span>{selectedEntity}</span>
              </div>
            </div>
          </div>
        ) : (
          <div className="inspector-empty">
            <span>Select an object in the scene</span>
          </div>
        )}
      </div>
    </div>
  );
}
