import { useEffect, useState } from "react";
import { api } from "../api";
import { Field, Toast, useToast } from "../ui";

export default function Contacts() {
  const [q, setQ] = useState("");
  const [rows, setRows] = useState<any[]>([]);
  const [total, setTotal] = useState(0);
  const [lists, setLists] = useState<any[]>([]);
  const [listId, setListId] = useState("");
  const [empty, setEmpty] = useState(false);
  const t = useToast();
  async function load() {
    const r = await api.contacts(q, 1);
    setRows(r.data || []);
    setTotal(r.meta?.total || 0);
    setEmpty((r.data || []).length === 0);
  }
  useEffect(() => {
    load().catch((e) => t.show(e.message));
    api.lists().then((ls) => { setLists(ls); if (ls[0]) setListId(ls[0].ID || ls[0].id); }).catch(() => {});
  }, []);
  async function onFile(f?: File) {
    if (!f) return;
    if (!listId) { t.show("请先选择客群"); return; }
    try {
      const r = await api.importFile(f, listId);
      t.show(`导入 ${r.job?.Imported ?? r.job?.imported ?? 0}，失败 ${r.job?.Failed ?? r.job?.failed ?? 0}`);
      load();
    } catch (e: any) { t.show(e.message); }
  }
  return (
    <div>
      <Toast text={t.msg} onClose={t.clear} />
      <h2 className="font-display text-3xl">客群</h2>
      <div className="mt-6 flex flex-col gap-3 md:flex-row md:items-end">
        <Field label="搜索">
          <input value={q} onChange={(e) => setQ(e.target.value)} className="rounded-2xl border border-line bg-ink px-3 py-2" />
        </Field>
        <button onClick={() => load()} className="rounded-full bg-brass px-4 py-2 text-sm">查找</button>
        <Field label="导入到">
          <select value={listId} onChange={(e) => setListId(e.target.value)} className="rounded-2xl border border-line bg-ink px-3 py-2">
            {lists.map((l) => <option key={l.ID || l.id} value={l.ID || l.id}>{l.Name || l.name}</option>)}
          </select>
        </Field>
        <label className="rounded-full border border-line px-4 py-2 text-sm cursor-pointer">
          导入 CSV / Excel
          <input type="file" accept=".csv,.xlsx" className="hidden" onChange={(e) => onFile(e.target.files?.[0])} />
        </label>
      </div>
      {empty ? (
        <div className="mt-16 text-center text-paper-dim">
          <p className="font-display text-2xl text-paper">信箱还是空的</p>
          <p className="mt-2">导入一份带 email / name 表头的名单，坏行会进入错误报告而不是毁掉整批。</p>
        </div>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-3xl border border-line">
          <table className="w-full text-left text-sm">
            <thead className="bg-ink-2 text-paper-dim"><tr><th className="p-3">邮箱</th><th>姓名</th><th>状态</th></tr></thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.ID || r.id} className="border-t border-line">
                  <td className="p-3 font-mono">{r.Email || r.email}</td>
                  <td>{r.Name || r.name}</td>
                  <td>{r.Status || r.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="mt-3 text-xs text-paper-dim">共 {total} 人</p>
    </div>
  );
}
