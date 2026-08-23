import { createRoot } from "react-dom/client";
import { useState } from "react";

const API = import.meta.env.VITE_API_BASE || "http://localhost:27482";

function App() {
  const t = new URLSearchParams(location.search).get("t") || "";
  const [done, setDone] = useState(false);
  const [err, setErr] = useState("");
  async function unsub() {
    try {
      const r = await fetch(API + "/api/v1/public/unsub", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: t }),
      });
      if (!r.ok) throw new Error("令牌无效或已失效");
      setDone(true);
    } catch (e: any) {
      setErr(e.message);
    }
  }
  return (
    <div style={{ minHeight: "100vh", background: "#14110e", color: "#f4efe6", fontFamily: "IBM Plex Sans, sans-serif", display: "flex", alignItems: "center", justifyContent: "center", padding: 24 }}>
      <div style={{ maxWidth: 420, border: "1px solid #2c261f", borderRadius: 28, padding: 40, background: "#1d1814" }}>
        <p style={{ letterSpacing: "0.3em", fontSize: 11, color: "#e8a25a" }}>LUMEN RELAY</p>
        <h1 style={{ fontFamily: "Fraunces, serif", fontSize: 36, margin: "8px 0" }}>不想再收到？</h1>
        <p style={{ color: "#c9bba8", lineHeight: 1.6 }}>退订即时生效。你的地址会进入隔离区，驿站不会再为了「再试一次」伤害域名信誉。</p>
        {done ? <p style={{ marginTop: 24, color: "#3d6b4f" }}>已退订。灯为别人留着。</p> : (
          <button onClick={unsub} style={{ marginTop: 28, width: "100%", border: 0, borderRadius: 999, padding: "12px 0", background: "#c45c26", color: "#f4efe6", cursor: "pointer" }}>确认退订</button>
        )}
        {err && <p style={{ color: "#9b2c1a", marginTop: 12 }}>{err}</p>}
      </div>
    </div>
  );
}

createRoot(document.getElementById("root")!).render(<App />);
