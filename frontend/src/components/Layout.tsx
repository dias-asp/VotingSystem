import { Link, Outlet, useNavigate } from "react-router-dom";
import { currentUser, tokens } from "../api/client";
import { auth } from "../api/endpoints";

export function Layout() {
  const user = currentUser();
  const nav = useNavigate();

  async function logout() {
    const refresh = tokens.refresh;
    tokens.clear();
    if (refresh) {
      try {
        await auth.logout(refresh);
      } catch {
        /* ignore */
      }
    }
    nav("/login");
  }

  return (
    <div className="min-h-screen flex flex-col">
      <header className="bg-slate-900 text-slate-100 px-6 py-3 flex items-center gap-6">
        <Link to="/" className="font-semibold">Voting System</Link>
        {user && <Link to="/polls" className="hover:underline">Polls</Link>}
        {user?.role === "admin" && <Link to="/admin" className="hover:underline">Admin</Link>}
        <span className="ml-auto text-sm text-slate-300">
          {user ? (
            <>
              <span className="mr-3">
                {user.email} <span className="opacity-70">({user.role})</span>
              </span>
              <button onClick={logout} className="px-2 py-1 bg-slate-700 hover:bg-slate-600 rounded">
                Logout
              </button>
            </>
          ) : (
            <>
              <Link to="/login" className="hover:underline mr-3">Login</Link>
              <Link to="/register" className="hover:underline">Register</Link>
            </>
          )}
        </span>
      </header>
      <main className="flex-1 p-6 max-w-4xl w-full mx-auto">
        <Outlet />
      </main>
    </div>
  );
}
