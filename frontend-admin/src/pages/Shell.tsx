import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useState } from "react";

const links = [
  { to: "/", label: "总览" },
  { to: "/contacts", label: "客群" },
  { to: "/templates", label: "信稿" },
  { to: "/campaigns", label: "活动" },
  { to: "/suppressions", label: "隔离区" },
];

export default function Shell() {
  const nav = useNavigate();
  const [open, setOpen] = useState(false);
  const role = localStorage.getItem("lumen_role") || "";
  return (
    <div className="flex min-h-screen">
      <aside className={`fixed z-20 h-full w-60 border-r border-line bg-ink-2 p-6 transition md:static ${open ? "translate-x-0" : "-translate-x-full md:translate-x-0"}`}>
        <p className="text-[10px] tracking-[0.35em] text-brass-2">LUMEN RELAY</p>
        <h1 className="mt-1 font-display text-2xl">流光驿站</h1>
        <nav className="mt-8 space-y-1">
          {links.map((l) => (
            <NavLink key={l.to} to={l.to} end={l.to === "/"} onClick={() => setOpen(false)}
              className={({ isActive }) => `block rounded-full px-4 py-2 text-sm ${isActive ? "bg-brass text-paper" : "text-paper-dim hover:bg-ink"}`}>
              {l.label}
            </NavLink>
          ))}
        </nav>
        <div className="absolute bottom-6 left-6 right-6 text-xs text-paper-dim">
          <p>角色 {role}</p>
          <button className="mt-2 underline" onClick={() => { localStorage.clear(); nav("/login"); }}>离开</button>
        </div>
      </aside>
      <div className="w-full md:ml-0">
        <header className="flex items-center justify-between border-b border-line px-4 py-3 md:hidden">
          <button onClick={() => setOpen(!open)} className="rounded-full border border-line px-3 py-1 text-sm">菜单</button>
          <span className="font-display">Lumen</span>
        </header>
        <main className="w-full p-6 md:p-10 paper-grain min-h-screen">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
