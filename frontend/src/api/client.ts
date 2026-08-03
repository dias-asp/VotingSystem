type FetchOpts = RequestInit & { auth?: boolean };

const TOKEN_KEY = "vs.access_token";
const REFRESH_KEY = "vs.refresh_token";
const ROLE_KEY = "vs.role";
const EMAIL_KEY = "vs.email";

export const tokens = {
  get access() {
    return localStorage.getItem(TOKEN_KEY);
  },
  get refresh() {
    return localStorage.getItem(REFRESH_KEY);
  },
  get role() {
    return localStorage.getItem(ROLE_KEY);
  },
  get email() {
    return localStorage.getItem(EMAIL_KEY);
  },
  set(access: string, refresh: string, role: string, email: string) {
    localStorage.setItem(TOKEN_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
    localStorage.setItem(ROLE_KEY, role);
    localStorage.setItem(EMAIL_KEY, email);
  },
  clear() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
    localStorage.removeItem(ROLE_KEY);
    localStorage.removeItem(EMAIL_KEY);
  },
};

function decodeJwt(token: string): { sub?: string; email?: string; role?: string; exp?: number } {
  try {
    const [, payload] = token.split(".");
    const padded = payload.replace(/-/g, "+").replace(/_/g, "/");
    return JSON.parse(atob(padded));
  } catch {
    return {};
  }
}

export function currentUser() {
  const access = tokens.access;
  if (!access) return null;
  const claims = decodeJwt(access);
  if (claims.exp && claims.exp * 1000 < Date.now()) return null;
  return {
    id: claims.sub ?? "",
    email: claims.email ?? tokens.email ?? "",
    role: claims.role ?? tokens.role ?? "user",
  };
}

export class ApiError extends Error {
  constructor(public status: number, public body: unknown) {
    super(typeof body === "object" && body && "error" in body ? String((body as { error: unknown }).error) : `HTTP ${status}`);
  }
}

export async function api<T>(path: string, opts: FetchOpts = {}): Promise<T> {
  const headers = new Headers(opts.headers);
  if (opts.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  if (opts.auth !== false) {
    const t = tokens.access;
    if (t) headers.set("Authorization", `Bearer ${t}`);
  }
  const res = await fetch(path, { ...opts, headers });
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  const body = text ? JSON.parse(text) : undefined;
  if (!res.ok) throw new ApiError(res.status, body);
  return body as T;
}
