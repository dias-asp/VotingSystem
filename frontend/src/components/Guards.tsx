import { Navigate, Outlet, useLocation } from "react-router-dom";
import { currentUser } from "../api/client";

export function RequireAuth() {
  const user = currentUser();
  const loc = useLocation();
  if (!user) return <Navigate to="/login" state={{ from: loc }} replace />;
  return <Outlet />;
}

export function RequireAdmin() {
  const user = currentUser();
  if (user?.role !== "admin") return <Navigate to="/polls" replace />;
  return <Outlet />;
}
