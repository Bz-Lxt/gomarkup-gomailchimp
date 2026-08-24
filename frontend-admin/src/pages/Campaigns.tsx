import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api";
import { Dialog, Field, Toast, useToast } from "../ui";

export default function Campaigns() {
  const [rows, setRows] = useState<any[]>([]);
  const [lists, setLists] = useState<any[]>([]);
  const [tpls, setTpls] = useState<any[]>([]);
  const [form, setForm] = useState({ name: "八月唤醒", from_name: "北极星", from_email: "hello@lumen.local", subject: "灯还亮着", list_id: "", template_ver_id: "", strategy: "immediate" });
  const [confirm, setConfirm] = useState<string>("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const t = useToast();
  async function load() {
    setRows(await api.campaigns());
    const ls = await api.lists();
    setLists(ls);
    const ts = await api.templates();
    setTpls(ts);
    if (!form.list_id && ls[0]) setForm((f) => ({ ...f, list_id: ls[0].ID || ls[0].id }));
  }
  useEffect(() => { load().catch((e) => t.show(e.message)); }, []);
  async function create() {
    const e: Record<string, string> = {};
    if (!form.name) e.name = "必填";
    if (!form.from_email.includes("@")) e.from_email = "邮箱格式不正确";
    if (!form.subject) e.subject = "必填";
    setErrors(e);
    if (Object.keys(e).length) { t.show("请修正表单"); return; }
    let ver = form.template_ver_id;
    if (!ver && tpls[0]) {
      const d = await api.getTemplate(tpls[0].ID || tpls[0].id);
      ver = d.version?.ID || d.version?.id;
    }
    try {
      await api.createCampaign({ ...form, template_ver_id: ver });
      t.show("活动已建立");
      load();
    } catch (err: any) { t.show(err.message); }
  }
  async function act(id: string, action: string) {
    try { await api.action(id, action); t.show("状态已更新"); load(); }
    catch (e: any) { t.show(e.message); }
  }
  return (
    <div>
      <Toast text={t.msg} onClose={t.clear} />
      <h2 className="font-display text-3xl">活动控制台</h2>
      <div className="mt-6 grid gap-3 md:grid-cols-3">
        <Field label="名称 *" error={errors.name}><input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="w-full rounded-2xl border border-line bg-ink px-3 py-2" /></Field>
        <Field label="发件人" ><input value={form.from_name} onChange={(e) => setForm({ ...form, from_name: e.target.value })} className="w-full rounded-2xl border border-line bg-ink px-3 py-2" /></Field>
        <Field label="From *" error={errors.from_email}><input value={form.from_email} onChange={(e) => setForm({ ...form, from_email: e.target.value })} className="w-full rounded-2xl border border-line bg-ink px-3 py-2" /></Field>
        <Field label="主题 *" error={errors.subject}><input value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })} className="w-full rounded-2xl border border-line bg-ink px-3 py-2" /></Field>
        <Field label="客群">
          <select value={form.list_id} onChange={(e) => setForm({ ...form, list_id: e.target.value })} className="w-full rounded-2xl border border-line bg-ink px-3 py-2">
            {lists.map((l) => <option key={l.ID || l.id} value={l.ID || l.id}>{l.Name || l.name}</option>)}
          </select>
        </Field>
        <Field label="策略">
          <select value={form.strategy} onChange={(e) => setForm({ ...form, strategy: e.target.value })} className="w-full rounded-2xl border border-line bg-ink px-3 py-2">
            <option value="immediate">立即发送</option>
            <option value="scheduled">定时发送</option>
            <option value="throttled">分批渐进</option>
          </select>
        </Field>
      </div>
      <button onClick={create} className="mt-4 rounded-full bg-brass px-5 py-2 text-sm">建立草稿</button>
      <div className="mt-8 overflow-x-auto rounded-3xl border border-line">
        <table className="w-full text-left text-sm">
          <thead className="bg-ink-2 text-paper-dim"><tr><th className="p-3">名称</th><th>状态</th><th>策略</th><th></th></tr></thead>
          <tbody>
            {rows.map((r) => {
              const id = r.ID || r.id;
              return (
                <tr key={id} className="border-t border-line">
                  <td className="p-3">{r.Name || r.name}</td>
                  <td>{r.Status || r.status}</td>
                  <td>{r.Strategy || r.strategy}</td>
                  <td className="space-x-2 p-3">
                    <button onClick={() => act(id, "start")} className="text-brass-2 underline">发送</button>
                    <button onClick={() => setConfirm(id)} className="text-paper-dim underline">暂停</button>
                    <Link to={`/funnel/${id}`} className="underline">漏斗</Link>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <Dialog open={!!confirm} title="暂停活动？" onClose={() => setConfirm("")} onConfirm={() => { act(confirm, "pause"); setConfirm(""); }}>
        在途任务会挂起，不会丢件。确认暂停？
      </Dialog>
    </div>
  );
}
