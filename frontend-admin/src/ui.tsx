import { useEffect, useState } from "react";

export function Toast({ text, onClose }: { text: string; onClose: () => void }) {
  useEffect(() => {
    const t = setTimeout(onClose, 5000);
    return () => clearTimeout(t);
  }, [onClose]);
  if (!text) return null;
  return (
    <div className="fixed right-4 top-4 z-50 flex items-start gap-3 rounded-2xl border border-line bg-ink-2 px-4 py-3 text-sm text-paper shadow-xl">
      <span>{text}</span>
      <button onClick={onClose} className="text-paper-dim hover:text-paper" aria-label="关闭">×</button>
    </div>
  );
}

export function Dialog({
  open, title, children, onClose, onConfirm, danger,
}: { open: boolean; title: string; children?: React.ReactNode; onClose: () => void; onConfirm?: () => void; danger?: boolean }) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-md rounded-3xl border border-line bg-ink-2 p-6">
        <h3 className="font-display text-xl">{title}</h3>
        <div className="mt-3 text-sm text-paper-dim">{children}</div>
        <div className="mt-6 flex justify-end gap-2">
          <button onClick={onClose} className="rounded-full border border-line px-4 py-2 text-sm">取消</button>
          {onConfirm && (
            <button onClick={onConfirm} className={`rounded-full px-4 py-2 text-sm text-paper ${danger ? "bg-rust" : "bg-brass"}`}>确认</button>
          )}
        </div>
      </div>
    </div>
  );
}

export function Field({ label, error, children }: { label: string; error?: string; children: React.ReactNode }) {
  return (
    <label className="block space-y-1">
      <span className="text-xs uppercase tracking-wider text-paper-dim">{label}</span>
      {children}
      {error && <span className="text-xs text-rust">{error}</span>}
    </label>
  );
}

export function useToast() {
  const [msg, setMsg] = useState("");
  return { msg, show: (t: string) => setMsg(t), clear: () => setMsg("") };
}
