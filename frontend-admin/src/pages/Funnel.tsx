import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import ReactECharts from "echarts-for-react";
import { api, getToken, sseURL } from "../api";

export default function Funnel() {
  const { id } = useParams();
  const [snap, setSnap] = useState<any>();
  const [live, setLive] = useState(true);
  useEffect(() => {
    if (!id) return;
    api.funnel(id).then(setSnap).catch(() => {});
    const es = new EventSource(sseURL(id) + `?access_token=${encodeURIComponent(getToken())}`);
    // EventSource cannot set Authorization; fall back to polling if SSE 401
    const iv = setInterval(() => { api.funnel(id!).then(setSnap).catch(() => {}); }, 2000);
    es.addEventListener("funnel", (e) => {
      try { setSnap(JSON.parse((e as MessageEvent).data)); setLive(true); } catch { /* ignore */ }
    });
    es.onerror = () => setLive(false);
    return () => { es.close(); clearInterval(iv); };
  }, [id]);
  void getToken;
  if (!snap) return <p className="text-paper-dim">漏斗点亮中…</p>;
  const sent = snap.sent || 0;
  const delivered = snap.delivered || sent;
  const opened = snap.unique_opened || 0;
  const clicked = snap.unique_click || 0;
  const bad = (snap.unsubscribed || 0) + (snap.complained || 0);
  const option = {
    backgroundColor: "transparent",
    tooltip: { trigger: "item", formatter: (p: any) => `${p.name}<br/>${p.value}` },
    series: [{
      type: "funnel",
      left: "10%", width: "80%",
      min: 0, max: Math.max(sent, 1),
      sort: "descending",
      gap: 8,
      label: { color: "#f4efe6", fontFamily: "Fraunces", fontSize: 16 },
      itemStyle: { borderColor: "#14110e", borderWidth: 2 },
      data: [
        { value: sent, name: "发送总数", itemStyle: { color: "#e8a25a" } },
        { value: delivered, name: "投递成功", itemStyle: { color: "#c45c26" } },
        { value: opened, name: "邮件打开", itemStyle: { color: "#3d6b4f" } },
        { value: clicked, name: "链接点击", itemStyle: { color: "#8a5a2b" } },
        { value: bad, name: "退订/投诉", itemStyle: { color: "#9b2c1a" } },
      ],
    }],
  };
  return (
    <div>
      <div className="flex items-end justify-between">
        <div>
          <p className="text-xs tracking-[0.3em] text-brass-2">{live ? "LIVE" : "RECONNECTING"}</p>
          <h2 className="font-display text-3xl">流式漏斗</h2>
        </div>
        <p className="text-xs text-paper-dim max-w-sm">打开分真实 / 机器。Gmail 图片代理与 Apple MPP 预取计入机器打开，不污染真实打开率。</p>
      </div>
      <div className="mt-6 grid gap-4 md:grid-cols-4">
        {[
          ["真实打开", snap.unique_opened],
          ["机器打开", snap.machine_open],
          ["合计打开", snap.opened],
          ["硬退信", snap.bounced],
        ].map(([k, v]) => (
          <div key={String(k)} className="rounded-3xl border border-line bg-ink-2 p-5">
            <p className="text-xs text-paper-dim">{k}</p>
            <p className="font-display text-3xl">{v || 0}</p>
          </div>
        ))}
      </div>
      <div className="mt-6 rounded-[32px] border border-line bg-ink-2 p-4">
        <ReactECharts option={option} style={{ height: 420 }} />
      </div>
    </div>
  );
}
