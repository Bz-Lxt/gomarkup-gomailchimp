import { Navigate, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { useEffect, useState } from "react";
import Login from "./pages/Login";
import Shell from "./pages/Shell";
import Home from "./pages/Home";
import Contacts from "./pages/Contacts";
import Builder from "./pages/Builder";
import Campaigns from "./pages/Campaigns";
import Funnel from "./pages/Funnel";
import Suppress from "./pages/Suppress";

export default function App() {
  const loc = useLocation();
  const nav = useNavigate();
  const [authed, setAuthed] = useState(!!localStorage.getItem("lumen_access"));
  useEffect(() => setAuthed(!!localStorage.getItem("lumen_access")), [loc]);
  if (!authed && loc.pathname !== "/login") return <Navigate to="/login" replace />;
  return (
    <Routes>
      <Route path="/login" element={<Login onOk={() => nav("/")} />} />
      <Route element={<Shell />}>
        <Route path="/" element={<Home />} />
        <Route path="/contacts" element={<Contacts />} />
        <Route path="/templates" element={<Builder />} />
        <Route path="/campaigns" element={<Campaigns />} />
        <Route path="/funnel/:id" element={<Funnel />} />
        <Route path="/suppressions" element={<Suppress />} />
      </Route>
    </Routes>
  );
}
