import { useState } from "react";

type LogLevel = "info" | "warn" | "error";

interface LogEntry {
  level: LogLevel;
  message: string;
  time: string;
}

const sampleLogs: LogEntry[] = [
  { level: "info", message: "GoEngineKenga IDE ready", time: "12:00:00" },
  { level: "info", message: "Project loaded: samples/hello", time: "12:00:01" },
  { level: "warn", message: "WebSocket not connected", time: "12:00:02" },
];

export function ConsolePanel() {
  const [logs] = useState<LogEntry[]>(sampleLogs);
  const [filter, setFilter] = useState<LogLevel | "all">("all");

  const filtered = filter === "all" ? logs : logs.filter((l) => l.level === filter);

  return (
    <div className="console-panel">
      <div className="panel-header">
        <span>Console</span>
        <div className="console-filters">
          <button className={filter === "all" ? "active" : ""} onClick={() => setFilter("all")}>All</button>
          <button className={filter === "info" ? "active" : ""} onClick={() => setFilter("info")}>Info</button>
          <button className={filter === "warn" ? "active" : ""} onClick={() => setFilter("warn")}>Warn</button>
          <button className={filter === "error" ? "active" : ""} onClick={() => setFilter("error")}>Error</button>
        </div>
      </div>
      <div className="console-output">
        {filtered.map((entry, i) => (
          <div key={i} className={`log-entry log-${entry.level}`}>
            <span className="log-time">[{entry.time}]</span>
            <span className="log-msg">{entry.message}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
