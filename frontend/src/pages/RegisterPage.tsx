import { FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { auth } from "../api/endpoints";
import { tokens, ApiError } from "../api/client";

export function RegisterPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const nav = useNavigate();

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await auth.register(email, password);
      const pair = await auth.login(email, password);
      const claims = JSON.parse(atob(pair.access_token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/")));
      tokens.set(pair.access_token, pair.refresh_token, claims.role ?? "user", claims.email ?? email);
      nav("/polls");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Registration failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="max-w-sm mx-auto bg-white rounded-lg shadow p-6">
      <h1 className="text-xl font-semibold mb-4">Register</h1>
      <form onSubmit={submit} className="space-y-3">
        <input
          className="w-full border rounded px-3 py-2"
          placeholder="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <input
          className="w-full border rounded px-3 py-2"
          placeholder="password (min 8 chars)"
          type="password"
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {error && <div className="text-sm text-red-600">{error}</div>}
        <button
          disabled={busy}
          className="w-full bg-slate-900 text-white py-2 rounded hover:bg-slate-700 disabled:opacity-50"
        >
          {busy ? "..." : "Create account"}
        </button>
      </form>
      <p className="text-sm text-slate-600 mt-4">
        Have an account? <Link to="/login" className="underline">Login</Link>
      </p>
      <p className="text-xs text-slate-500 mt-2">
        Tip: register with the email configured as <code>ADMIN_EMAIL</code> on auth_service to get admin rights.
      </p>
    </div>
  );
}
