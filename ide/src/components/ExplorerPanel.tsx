import { useState } from "react";

interface FileNode {
  name: string;
  type: "file" | "folder";
  children?: FileNode[];
}

const defaultTree: FileNode[] = [
  { name: "project", type: "folder", children: [
    { name: "project.kenga.json", type: "file" },
    { name: "main.go", type: "file" },
    { name: "assets", type: "folder", children: [
      { name: "triangle.gltf", type: "file" },
      { name: "helmet", type: "folder", children: [
        { name: "DamagedHelmet.gltf", type: "file" },
        { name: "Default_normal.jpg", type: "file" },
      ]},
    ]},
    { name: "scenes", type: "folder", children: [
      { name: "main.scene.json", type: "file" },
      { name: "physics_test.scene.json", type: "file" },
    ]},
    { name: "scripts", type: "folder", children: [
      { name: "game", type: "folder", children: [
        { name: "main.go", type: "file" },
      ]},
    ]},
  ]},
];

function TreeNode({ node, depth = 0, onSelect }: { node: FileNode; depth?: number; onSelect?: (path: string) => void }) {
  const [open, setOpen] = useState(depth < 2);
  const isFolder = node.type === "folder";

  return (
    <div className="tree-node">
      <div
        className={`tree-item ${isFolder ? "folder" : "file"}`}
        style={{ paddingLeft: 8 + depth * 16 }}
        onClick={() => {
          if (isFolder) setOpen(!open);
          else onSelect?.(node.name);
        }}
      >
        <span className="tree-icon">
          {isFolder ? (open ? "▼" : "▶") : "📄"}
        </span>
        {node.name}
      </div>
      {isFolder && open && node.children?.map((child, i) => (
        <TreeNode key={i} node={child} depth={depth + 1} onSelect={onSelect} />
      ))}
    </div>
  );
}

interface ExplorerPanelProps {
  onFileSelect?: (path: string) => void;
}

export function ExplorerPanel({ onFileSelect }: ExplorerPanelProps) {
  const [activeTab, setActiveTab] = useState<"explorer" | "search">("explorer");

  return (
    <div className="explorer-panel">
      <div className="panel-tabs">
        <button className={activeTab === "explorer" ? "active" : ""} onClick={() => setActiveTab("explorer")}>
          Explorer
        </button>
        <button className={activeTab === "search" ? "active" : ""} onClick={() => setActiveTab("search")}>
          Search
        </button>
      </div>
      <div className="panel-content">
        {activeTab === "explorer" && (
          <div className="file-tree">
            {defaultTree.map((node, i) => (
              <TreeNode key={i} node={node} onSelect={onFileSelect} />
            ))}
          </div>
        )}
        {activeTab === "search" && (
          <div className="search-placeholder">
            <input type="text" placeholder="Search in files..." />
          </div>
        )}
      </div>
    </div>
  );
}
