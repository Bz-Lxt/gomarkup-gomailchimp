import { useState } from "react";
import { api } from "../api";
import { Field, Toast, useToast } from "../ui";

export default function Login({ onOk }: { onOk: () => void }) {
  const [email, setEmail] = useState("owner@lumen.local");
  const [password, setPassword] = useState("Owner123!");
  const [err, setErr] = useState("");
  const t = useToast();
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setErr("");
    if (!email.includes("@")) { setErr("请输入有效邮箱"); t.show("请修正表单"); return; }
    if (password.length < 6) { setErr("密码过短"); t.show("请修正表单"); return; }
    try {
      const r = await api.login(email, password);
      localStorage.setItem("lumen_access", r.tokens.access_token);
      localStorage.setItem("lumen_refresh", r.tokens.refresh_token);
      localStorage.setItem("lumen_role", r.user.role);
      onOk();
    } catch (e: any) {
      t.show(e.message || "登录失败");
    }
  }
  return (
    <div className="paper-grain flex min-h-screen items-center justify-center px-4">
      <Toast text={t.msg} onClose={t.clear} />
      <form onSubmit={submit} className="w-full max-w-md rounded-[32px] border border-line bg-ink-2 p-10 shadow-2xl">
        <p className="text-xs tracking-[0.3em] text-brass-2">LUMEN RELAY</p>
        <h1 className="mt-2 font-display text-4xl">流光驿站</h1>
        <p className="mt-2 text-sm text-paper-dim">把信送出去，且不被夜色吞没。</p>
        <div className="mt-8 space-y-4">
          <Field label="邮箱 *" error={err && !email.includes("@") ? err : ""}>
            <input value={email} onChange={(e) => setEmail(e.target.value)} className="w-full rounded-2xl border border-line bg-ink px-4 py-3 outline-none focus:border-brass" />
          </Field>
          <Field label="密码 *">
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} className="w-full rounded-2xl border border-line bg-ink px-4 py-3 outline-none focus:border-brass" />
          </Field>
        </div>
        <button className="mt-8 w-full rounded-full bg-brass py-3 font-medium text-paper">进入驿站</button>
        <p className="mt-4 text-center text-xs text-paper-dim">试用 owner@lumen.local / Owner123!</p>
      </form>
    </div>
  );
}
