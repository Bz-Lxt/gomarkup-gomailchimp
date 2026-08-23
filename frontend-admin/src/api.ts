const API = import.meta.env.VITE_API_BASE || "http://localhost:27482";

export type Tokens = { access_token: string; refresh_token: string; expires_in: number };

export function getToken() {
  return localStorage.getItem("lumen_access") || "";
}

async function req<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = { ...(init.headers as Record<string, string>) };
  if (!(init.body instanceof FormData)) headers["Content-Type"] = "application/json";
  const t = getToken();
  if (t) headers.Authorization = `Bearer ${t}`;
  const res = await fetch(API + path, { ...init, headers });
  const json = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(json?.error?.message || res.statusText);
  return (json.data ?? json) as T;
}

export const api = {
  login: (email: string, password: string) =>
    req<{ tokens: Tokens; user: { email: string; role: string; tenant_id: string; id: string } }>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),
  me: () => req<{ user: { email: string; role: string }; tenant: { name: string } }>("/api/v1/me"),
  contacts: (q = "", page = 1) => req<{ /* overwritten */ }>(`/api/v1/contacts?q=${encodeURIComponent(q)}&page=${page}&per_page=20`).then(async () => {
    const t = getToken();
    const res = await fetch(`${API}/api/v1/contacts?q=${encodeURIComponent(q)}&page=${page}&per_page=20`, {
      headers: { Authorization: `Bearer ${t}` },
    });
    return res.json();
  }),
  lists: () => req<any[]>("/api/v1/lists"),
  createList: (name: string) => req("/api/v1/lists", { method: "POST", body: JSON.stringify({ name }) }),
  importFile: async (file: File, listId: string) => {
    const fd = new FormData();
    fd.append("file", file);
    fd.append("list_id", listId);
    const t = getToken();
    const res = await fetch(API + "/api/v1/contacts/import", {
      method: "POST",
      headers: { Authorization: `Bearer ${t}` },
      body: fd,
    });
    const json = await res.json();
    if (!res.ok) throw new Error(json?.error?.message || "导入失败");
    return json.data;
  },
  templates: () => req<any[]>("/api/v1/templates"),
  getTemplate: (id: string) => req<any>(`/api/v1/templates/${id}`),
  saveTemplate: (body: any) => req("/api/v1/templates", { method: "POST", body: JSON.stringify(body) }),
  campaigns: () => req<any[]>("/api/v1/campaigns"),
  createCampaign: (body: any) => req("/api/v1/campaigns", { method: "POST", body: JSON.stringify(body) }),
  campaign: (id: string) => req<any>(`/api/v1/campaigns/${id}`),
  action: (id: string, action: string) =>
    req(`/api/v1/campaigns/${id}/action`, { method: "POST", body: JSON.stringify({ action }) }),
  funnel: (id: string) => req<any>(`/api/v1/campaigns/${id}/funnel`),
  channels: () => req<any[]>("/api/v1/channels"),
  suppressions: () => req<any[]>("/api/v1/suppressions"),
  pipe: () => req<any>("/api/v1/pipeline/stats"),
};

export function sseURL(id: string) {
  return `${API}/api/v1/campaigns/${id}/stream`;
}
