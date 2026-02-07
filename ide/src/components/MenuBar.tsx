import { Command } from "@tauri-apps/plugin-shell";

export function MenuBar() {
  const handleNew = () => console.log("File > New");
  const handleOpen = () => console.log("File > Open");
  const handleSave = () => console.log("File > Save");
  const handleRun = async () => {
    try {
      await Command.create("kenga", ["run", "--project", ".", "--scene", "scenes/main.scene.json"]).execute();
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <header className="menu-bar">
      <div className="menu-items">
        <div className="menu-item">
          <span>File</span>
          <div className="menu-dropdown">
            <div onClick={handleNew}>New Project</div>
            <div onClick={handleOpen}>Open Project</div>
            <div onClick={handleSave}>Save</div>
            <div className="sep" />
            <div>Exit</div>
          </div>
        </div>
        <div className="menu-item">
          <span>Edit</span>
          <div className="menu-dropdown">
            <div>Undo</div>
            <div>Redo</div>
            <div className="sep" />
            <div>Cut</div>
            <div>Copy</div>
            <div>Paste</div>
          </div>
        </div>
        <div className="menu-item">
          <span>View</span>
          <div className="menu-dropdown">
            <div>Explorer</div>
            <div>Inspector</div>
            <div>Console</div>
            <div className="sep" />
            <div>Full Screen</div>
          </div>
        </div>
        <div className="menu-item">
          <span>Run</span>
          <div className="menu-dropdown">
            <div onClick={handleRun}>Run Scene</div>
            <div>Build</div>
            <div>Stop</div>
          </div>
        </div>
        <div className="menu-item">
          <span>Window</span>
          <div className="menu-dropdown">
            <div>Reset Layout</div>
            <div>Zoom In</div>
            <div>Zoom Out</div>
          </div>
        </div>
        <div className="menu-item">
          <span>Help</span>
          <div className="menu-dropdown">
            <div>Documentation</div>
            <div>About</div>
          </div>
        </div>
      </div>
      <div className="menu-title">GoEngineKenga IDE</div>
    </header>
  );
}
