import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import { BrowserRouter, Routes, Route } from "react-router";
import Home from "./pages/Home.tsx";
import WorkflowsGet from "./pages/workflows/Get.tsx";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<App />}>
          <Route index element={<Home />} />
          <Route path="workflows/:id" element={<WorkflowsGet />} />
        </Route>
      </Routes>
    </BrowserRouter>
  </StrictMode>
);
