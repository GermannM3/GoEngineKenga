import { useEffect, useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";
import { MenuBar } from "./components/MenuBar";
import { CommandLine } from "./components/CommandLine";
import { ExplorerPanel } from "./components/ExplorerPanel";
import { EditorTabs } from "./components/EditorTabs";
import { Viewport } from "./components/Viewport";
import { InspectorPanel } from "./components/InspectorPanel";
import { ConsolePanel } from "./components/ConsolePanel";
import "./App.css";

function App() {
  const [activeView, setActiveView] = useState<"editor" | "viewport">("editor");
  const [selectedEntity] = useState<string | null>(null);
  const [projectPath] = useState<string>(".");

  useEffect(() => {
    let unlisten: (() => void) | undefined;
    getCurrentWindow()
      .onDragDropEvent((event) => {
        if (event.payload.type === "drop" && event.payload.paths?.length) {
          const ext = (p: string) => p.toLowerCase().endsWith(".gltf") || p.toLowerCase().endsWith(".glb");
          const valid = event.payload.paths.filter(ext);
          if (valid.length) {
            invoke<string>("import_dropped_assets", {
              paths: valid,
              projectDir: projectPath,
            })
              .then((msg) => console.log(msg))
              .catch((e) => console.error("Import failed:", e));
          }
        }
      })
      .then((fn) => { unlisten = fn; })
      .catch(console.error);
    return () => unlisten?.();
  }, [projectPath]);

  const handleCommand = (cmd: string) => {
    console.log("Command:", cmd);
  };

  return (
    <div className="app">
      <MenuBar />

      <div className="main-area">
        <PanelGroup direction="vertical">
          <Panel defaultSize={85} minSize={50}>
            <PanelGroup direction="horizontal">
              <Panel defaultSize={18} minSize={12} maxSize={35}>
                <ExplorerPanel onFileSelect={(path) => console.log("Open:", path)} />
              </Panel>
              <PanelResizeHandle className="resize-handle horizontal" />

              <Panel defaultSize={55} minSize={30}>
                <div className="center-area">
                  <div className="center-tabs">
                    <button
                      className={activeView === "editor" ? "active" : ""}
                      onClick={() => setActiveView("editor")}
                    >
                      Editor
                    </button>
                    <button
                      className={activeView === "viewport" ? "active" : ""}
                      onClick={() => setActiveView("viewport")}
                    >
                      Scene
                    </button>
                  </div>
                  <div className="center-content">
                    {activeView === "editor" && <EditorTabs />}
                    {activeView === "viewport" && <Viewport />}
                  </div>
                </div>
              </Panel>
              <PanelResizeHandle className="resize-handle horizontal" />

              <Panel defaultSize={22} minSize={15} maxSize={35}>
                <InspectorPanel selectedEntity={selectedEntity} />
              </Panel>
            </PanelGroup>
          </Panel>
          <PanelResizeHandle className="resize-handle vertical" />
          <Panel defaultSize={15} minSize={10} maxSize={40}>
            <div className="bottom-area">
              <CommandLine onCommand={handleCommand} />
              <ConsolePanel />
            </div>
          </Panel>
        </PanelGroup>
      </div>

      <footer className="status-bar">
        <span>Ready</span>
        <span>Go 1.22+</span>
        <span>Ln 1, Col 1</span>
      </footer>
    </div>
  );
}

export default App;
