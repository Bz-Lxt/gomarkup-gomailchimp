import { useEffect, useState } from "react";
import {
  DndContext, DragEndEvent, PointerSensor, useSensor, useSensors, useDraggable, useDroppable,
} from "@dnd-kit/core";
import { SortableContext, arrayMove, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { api } from "../api";
import { Field, Toast, useToast } from "../ui";

type Block = {
  id: string; type: "text" | "image" | "button" | "divider";
  html?: string; align?: string; color?: string; fontSize?: number;
  src?: string; alt?: string; label?: string; url?: string; bg?: string; padding?: number;
};

const palette = [
  { type: "text", label: "文本" },
  { type: "image", label: "图片" },
  { type: "button", label: "按钮" },
  { type: "divider", label: "分割线" },
];

function PalItem({ type, label }: { type: string; label: string }) {
  const { attributes, listeners, setNodeRef, transform } = useDraggable({ id: "pal-" + type, data: { from: "palette", type } });
  const style = transform ? { transform: `translate(${transform.x}px,${transform.y}px)` } : undefined;
  return (
    <div ref={setNodeRef} style={style} {...listeners} {...attributes}
      className="cursor-grab rounded-2xl border border-line bg-ink px-3 py-2 text-sm text-paper-dim">
      {label}
    </div>
  );
}

function SortItem({ b, selected, onPick }: { b: Block; selected: boolean; onPick: () => void }) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: b.id });
  const style = { transform: CSS.Transform.toString(transform), transition };
  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners} onClick={onPick}
      className={`mb-2 rounded-xl border p-3 ${selected ? "border-brass" : "border-line"}`}>
      {b.type === "text" && <div style={{ color: b.color, fontSize: b.fontSize, textAlign: (b.align as any) }} dangerouslySetInnerHTML={{ __html: b.html || "" }} />}
      {b.type === "image" && <img src={b.src} alt={b.alt} className="max-w-full" />}
      {b.type === "button" && <div className="text-center"><span className="inline-block rounded-full px-5 py-2 text-sm text-paper" style={{ background: b.bg }}>{b.label}</span></div>}
      {b.type === "divider" && <hr style={{ borderColor: b.color }} />}
    </div>
  );
}

function fresh(type: string): Block {
  const id = Math.random().toString(36).slice(2);
  if (type === "text") return { id, type: "text", html: "Hi, {{ .UserName }}", align: "left", color: "#2b2118", fontSize: 16, padding: 16 };
  if (type === "image") return { id, type: "image", src: "https://picsum.photos/seed/lumen/560/200", alt: "hero", align: "center", padding: 8 };
  if (type === "button") return { id, type: "button", label: "回到书桌", url: "https://example.com/back", bg: "#c45c26", align: "center", padding: 16 };
  return { id, type: "divider", color: "#e6dccb", padding: 8 };
}

export default function Builder() {
  const [name, setName] = useState("21 日唤醒");
  const [subject, setSubject] = useState("Hi, {{ .UserName }}，灯还亮着");
  const [blocks, setBlocks] = useState<Block[]>([fresh("text"), fresh("button")]);
  const [sel, setSel] = useState<string>("");
  const [history, setHistory] = useState<Block[][]>([]);
  const t = useToast();
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));
  const { setNodeRef } = useDroppable({ id: "canvas" });

  useEffect(() => {
    api.templates().then((list) => {
      const id = list?.[0]?.ID || list?.[0]?.id;
      if (!id) return;
      api.getTemplate(id).then((d) => {
        const ast = typeof d.version?.AST === "string" ? JSON.parse(d.version.AST) : (d.version?.AST || d.version?.ast);
        if (ast?.blocks) setBlocks(ast.blocks);
        if (d.template?.Name) setName(d.template.Name);
        if (d.version?.Subject) setSubject(d.version.Subject);
      }).catch(() => {});
    }).catch(() => {});
  }, []);

  function pushHist(next: Block[]) {
    setHistory((h) => [...h.slice(-20), blocks]);
    setBlocks(next);
  }
  function onDragEnd(e: DragEndEvent) {
    const { active, over } = e;
    if (!over) return;
    if (String(active.id).startsWith("pal-")) {
      const type = String(active.data.current?.type);
      pushHist([...blocks, fresh(type)]);
      return;
    }
    if (active.id !== over.id) {
      const old = blocks.findIndex((b) => b.id === active.id);
      const neu = blocks.findIndex((b) => b.id === over.id);
      if (old >= 0 && neu >= 0) pushHist(arrayMove(blocks, old, neu));
    }
  }
  const cur = blocks.find((b) => b.id === sel);
  function patch(p: Partial<Block>) {
    pushHist(blocks.map((b) => (b.id === sel ? { ...b, ...p } : b)));
  }
  async function save() {
    if (!name.trim() || !subject.trim()) { t.show("名称与主题为必填"); return; }
    try {
      await api.saveTemplate({ name, subject, ast: { width: 600, background: "#f4efe6", blocks } });
      t.show("信稿已存为新版本");
    } catch (e: any) { t.show(e.message); }
  }
  return (
    <div>
      <Toast text={t.msg} onClose={t.clear} />
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="font-display text-3xl">信稿工坊</h2>
          <p className="text-sm text-paper-dim">拖入四类组件。占位符按 Go template 渲染，未知变量变空串。</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => { const prev = history.pop(); if (prev) { setBlocks(prev); setHistory([...history]); } }} className="rounded-full border border-line px-4 py-2 text-sm">撤销</button>
          <button onClick={save} className="rounded-full bg-brass px-4 py-2 text-sm">保存版本</button>
        </div>
      </div>
      <div className="mt-4 grid gap-3 md:grid-cols-2">
        <Field label="模板名 *"><input value={name} onChange={(e) => setName(e.target.value)} className="w-full rounded-2xl border border-line bg-ink px-3 py-2" /></Field>
        <Field label="主题 *"><input value={subject} onChange={(e) => setSubject(e.target.value)} className="w-full rounded-2xl border border-line bg-ink px-3 py-2" /></Field>
      </div>
      <DndContext sensors={sensors} onDragEnd={onDragEnd}>
        <div className="mt-6 grid gap-6 lg:grid-cols-[200px_1fr_260px]">
          <div className="space-y-2">
            <p className="text-xs uppercase tracking-wider text-paper-dim">组件库</p>
            {palette.map((p) => <PalItem key={p.type} {...p} />)}
            <button onClick={() => cur && patch({ html: (cur.html || "") + " {{ .UserName }}" })} className="mt-4 w-full rounded-full border border-line px-3 py-2 text-xs">插入 {"{{ .UserName }}"}</button>
          </div>
          <div ref={setNodeRef} className="rounded-[28px] border border-line bg-[#f4efe6] p-6 text-[#2b2118] min-h-[480px]">
            <p className="mb-4 text-center text-xs text-[#8a8175]">600px · 羊皮纸画布</p>
            <SortableContext items={blocks.map((b) => b.id)} strategy={verticalListSortingStrategy}>
              {blocks.map((b) => <SortItem key={b.id} b={b} selected={sel === b.id} onPick={() => setSel(b.id)} />)}
            </SortableContext>
            {blocks.length === 0 && <p className="text-center text-sm text-[#8a8175]">把左侧组件拖进来</p>}
          </div>
          <div className="space-y-3 rounded-3xl border border-line bg-ink-2 p-4">
            <p className="text-xs uppercase tracking-wider text-paper-dim">属性</p>
            {!cur && <p className="text-sm text-paper-dim">点选画布中的一块</p>}
            {cur?.type === "text" && (
              <>
                <Field label="正文"><textarea value={cur.html} onChange={(e) => patch({ html: e.target.value })} className="w-full rounded-xl border border-line bg-ink px-2 py-1 text-sm" rows={4} /></Field>
                <Field label="字号"><input type="number" value={cur.fontSize} onChange={(e) => patch({ fontSize: Number(e.target.value) })} className="w-full rounded-xl border border-line bg-ink px-2 py-1" /></Field>
                <Field label="颜色"><input value={cur.color} onChange={(e) => patch({ color: e.target.value })} className="w-full rounded-xl border border-line bg-ink px-2 py-1" /></Field>
              </>
            )}
            {cur?.type === "image" && (
              <>
                <Field label="图片 URL"><input value={cur.src} onChange={(e) => patch({ src: e.target.value })} className="w-full rounded-xl border border-line bg-ink px-2 py-1" /></Field>
                <Field label="Alt"><input value={cur.alt} onChange={(e) => patch({ alt: e.target.value })} className="w-full rounded-xl border border-line bg-ink px-2 py-1" /></Field>
              </>
            )}
            {cur?.type === "button" && (
              <>
                <Field label="文案"><input value={cur.label} onChange={(e) => patch({ label: e.target.value })} className="w-full rounded-xl border border-line bg-ink px-2 py-1" /></Field>
                <Field label="链接"><input value={cur.url} onChange={(e) => patch({ url: e.target.value })} className="w-full rounded-xl border border-line bg-ink px-2 py-1" /></Field>
                <Field label="按钮色"><input value={cur.bg} onChange={(e) => patch({ bg: e.target.value })} className="w-full rounded-xl border border-line bg-ink px-2 py-1" /></Field>
              </>
            )}
            {cur && <button onClick={() => pushHist(blocks.filter((b) => b.id !== sel))} className="rounded-full border border-rust px-3 py-1 text-xs text-rust">删除</button>}
          </div>
        </div>
      </DndContext>
    </div>
  );
}
