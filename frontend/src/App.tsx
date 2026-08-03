import { Navigate, Route, Routes } from "react-router-dom";
import { Layout } from "./components/Layout";
import { LoginPage } from "./pages/LoginPage";
import { RegisterPage } from "./pages/RegisterPage";
import { PollsListPage } from "./pages/PollsListPage";
import { PollDetailsPage } from "./pages/PollDetailsPage";
import { AdminPage } from "./pages/AdminPage";
import { RequireAuth, RequireAdmin } from "./components/Guards";

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route element={<RequireAuth />}>
          <Route path="/" element={<Navigate to="/polls" replace />} />
          <Route path="/polls" element={<PollsListPage />} />
          <Route path="/polls/:id" element={<PollDetailsPage />} />
          <Route element={<RequireAdmin />}>
            <Route path="/admin" element={<AdminPage />} />
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
