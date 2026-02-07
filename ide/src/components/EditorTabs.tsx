import { useState } from "react";
import Editor from "@monaco-editor/react";

interface Tab {
  id: string;
  title: string;
  language: string;
  content: string;
}

const initialTabs: Tab[] = [
  {
    id: "1",
    title: "main.go",
    language: "go",
    content: `// GoEngineKenga script
package main

func Update(dt float32) {
    // Your game logic
}
`,
  },
  {
    id: "2",
    title: "scene.json",
    language: "json",
    content: `{
  "entities": [],
  "camera": { "pos": [0, 5, 10], "target": [0, 0, 0] }
}
`,
  },
];

interface EditorTabsProps {
  onFileSelect?: (path: string) => void;
}

export function EditorTabs(_props: EditorTabsProps) {
  const [tabs, setTabs] = useState(initialTabs);
  const [activeId, setActiveId] = useState(tabs[0]?.id ?? "");
  const [content, setContent] = useState<Record<string, string>>(
    Object.fromEntries(tabs.map((t) => [t.id, t.content]))
  );

  const activeTab = tabs.find((t) => t.id === activeId);

  const closeTab = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const idx = tabs.findIndex((t) => t.id === id);
    if (idx < 0) return;
    const next = idx > 0 ? tabs[idx - 1].id : tabs[idx + 1]?.id;
    setTabs((t) => t.filter((x) => x.id !== id));
    setActiveId(next ?? "");
  };

  return (
    <div className="editor-tabs">
      <div className="tab-bar">
        {tabs.map((tab) => (
          <div
            key={tab.id}
            className={`tab ${activeId === tab.id ? "active" : ""}`}
            onClick={() => setActiveId(tab.id)}
          >
            <span>{tab.title}</span>
            <button className="tab-close" onClick={(e) => closeTab(tab.id, e)}>×</button>
          </div>
        ))}
        <div className="tab-add" title="New file">+</div>
      </div>
      <div className="editor-container">
        {activeTab && (
          <Editor
            height="100%"
            language={activeTab.language}
            value={content[activeTab.id] ?? activeTab.content}
            onChange={(v) => setContent((c) => ({ ...c, [activeTab.id]: v ?? "" }))}
            theme="vs-dark"
            options={{
              minimap: { enabled: true },
              fontSize: 14,
              wordWrap: "on",
              padding: { top: 8 },
            }}
          />
        )}
      </div>
    </div>
  );
}
