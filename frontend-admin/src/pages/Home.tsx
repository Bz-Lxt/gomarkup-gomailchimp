import { useEffect, useState } from "react";
import { api } from "../api";
import { Link } from "react-router-dom";

export default function Home() {
  const [me, setMe] = useState<any>();
  const [pipe, setPipe] = useState<any>();
  const [err, setErr] = useState("");
  useEffect(() => {
    api.me().then(setMe).catch((e) => setErr(e.message));
    api.pipe().then(setPipe).catch(() => {});
  }, []);
  if (err) return <p className="text-rust">{err}</p>;
  if (!me) return <p className="text-paper-dim">点亮驿站灯火…</p>;
  return (
    <div>
      <p className="text-xs tracking-[0.3em] text-brass-2">TONIGHT</p>
      <h2 className="font-display text-4xl">{me.tenant?.name || "租户"}</h2>
      <p className="mt-2 text-paper-dim">限速是潮汐闸，不是刹车。Gmail 触顶时 Outlook 仍全速出港。</p>
      <div className="mt-8 grid gap-4 md:grid-cols-3">
        {[
          ["发送队列", pipe?.send ?? "—"],
          ["延迟队列", pipe?.delay ?? "—"],
          ["死信", pipe?.dlq ?? "—"],
        ].map(([k, v]) => (
          <div key={k} className="rounded-3xl border border-line bg-ink-2 p-6">
            <p className="text-xs text-paper-dim">{k}</p>
            <p className="mt-2 font-display text-4xl">{v}</p>
          </div>
        ))}
      </div>
      <div className="mt-8 flex flex-wrap gap-3">
        <Link to="/templates" className="rounded-full bg-brass px-5 py-2 text-sm">写一封信</Link>
        <Link to="/campaigns" className="rounded-full border border-line px-5 py-2 text-sm">发起活动</Link>
      </div>
    </div>
  );
}
