import { useState } from "react";

interface CommandLineProps {
  onCommand?: (cmd: string) => void;
  placeholder?: string;
}

export function CommandLine({ onCommand, placeholder = "Command:" }: CommandLineProps) {
  const [value, setValue] = useState("");
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);

  const submit = () => {
    const cmd = value.trim();
    if (!cmd) return;
    setHistory((h) => [...h, cmd]);
    setHistoryIndex(-1);
    setValue("");
    onCommand?.(cmd);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      submit();
    } else if (e.key === "ArrowUp" && history.length > 0) {
      e.preventDefault();
      const idx = historyIndex < 0 ? history.length - 1 : Math.max(0, historyIndex - 1);
      setHistoryIndex(idx);
      setValue(history[idx]);
    } else if (e.key === "ArrowDown" && historyIndex >= 0) {
      e.preventDefault();
      const idx = historyIndex + 1;
      if (idx >= history.length) {
        setHistoryIndex(-1);
        setValue("");
      } else {
        setHistoryIndex(idx);
        setValue(history[idx]);
      }
    }
  };

  return (
    <div className="command-line">
      <span className="command-prompt">{placeholder}</span>
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="kenga run, load_model, set_camera..."
        spellCheck={false}
        autoComplete="off"
      />
    </div>
  );
}
