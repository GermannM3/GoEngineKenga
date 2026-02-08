import { useEffect, useRef, useState } from "react";

const WS_URL = "ws://127.0.0.1:7777/ws";

export function Viewport() {
  const [connected, setConnected] = useState(false);
  const [frame, setFrame] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectRef = useRef<number | null>(null);

  useEffect(() => {
    let closed = false;

    const connect = () => {
      if (closed) return;
      try {
        const ws = new WebSocket(WS_URL);
        wsRef.current = ws;

        ws.onopen = () => {
          setConnected(true);
          setError(null);
          ws.send(JSON.stringify({ cmd: "subscribe_viewport" }));
        };

        ws.onmessage = (ev) => {
          try {
            const data = JSON.parse(ev.data as string);
            if (data.event === "viewport_frame") {
              const d = data.data;
              const f = typeof d === "string" ? JSON.parse(d)?.frame : d?.frame;
              if (f) setFrame(f);
            }
          } catch {
            // ignore parse errors
          }
        };

        ws.onclose = () => {
          setConnected(false);
          wsRef.current = null;
          if (!closed && reconnectRef.current == null) {
            reconnectRef.current = window.setTimeout(() => {
              reconnectRef.current = null;
              connect();
            }, 2000);
          }
        };

        ws.onerror = () => {
          setError("Connection failed");
        };
      } catch (e) {
        setError(String(e));
        setConnected(false);
      }
    };

    connect();

    return () => {
      closed = true;
      if (reconnectRef.current != null) {
        clearTimeout(reconnectRef.current);
        reconnectRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.send(JSON.stringify({ cmd: "unsubscribe_viewport" }));
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, []);

  return (
    <div className="viewport">
      <div className="viewport-toolbar">
        <span className="viewport-title">Scene</span>
        <span
          className={`viewport-status ${connected ? "connected" : ""}`}
          title={connected ? "Connected" : "Disconnected"}
        >
          {connected ? "●" : "○"}
        </span>
        <div className="viewport-controls">
          <button title="Perspective">Persp</button>
          <button title="Wireframe">Wire</button>
          <button title="Shaded">Shaded</button>
          <button title="Grid">Grid</button>
        </div>
      </div>
      <div className="viewport-canvas">
        {frame ? (
          <img src={`data:image/png;base64,${frame}`} alt="Scene preview" />
        ) : (
          <div className="viewport-placeholder">
            <span>3D Viewport</span>
            <small>
              {error
                ? error
                : connected
                  ? "Waiting for frames…"
                  : "Run: kenga run --project . (WebSocket on :7777)"}
            </small>
          </div>
        )}
      </div>
    </div>
  );
}
