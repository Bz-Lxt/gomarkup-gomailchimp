import { useEffect, useState } from "react";
import { api } from "../api";

export default function Suppress() {
  const [rows, setRows] = useState<any[]>([]);
  useEffect(() => { api.suppressions().then(setRows).catch(() => {}); }, []);
  if (!rows.length) {
    return (
      <div>
        <h2 className="font-display text-3xl">隔离区</h2>
        <p className="mt-10 text-paper-dim">还没有死信。硬退信与投诉会自动关进这里，下次群发必跳过。</p>
      </div>
    );
  }
  return (
    <div>
      <h2 className="font-display text-3xl">隔离区</h2>
      <div className="mt-6 overflow-x-auto rounded-3xl border border-line">
        <table className="w-full text-left text-sm">
          <thead className="bg-ink-2 text-paper-dim"><tr><th className="p-3">邮箱</th><th>原因</th><th>来源</th><th>时间</th></tr></thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.ID || r.id} className="border-t border-line">
                <td className="p-3 font-mono">{r.Email || r.email}</td>
                <td>{r.Reason || r.reason}</td>
                <td>{r.Source || r.source}</td>
                <td>{r.CreatedAt || r.created_at}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
