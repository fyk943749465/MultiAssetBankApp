import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AppLayout } from "./layouts/AppLayout";
import { BankPage } from "./pages/BankPage";
import { CrowdfundingPage } from "./pages/CrowdfundingPage";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<Navigate to="/bank" replace />} />
          <Route path="bank" element={<BankPage />} />
          <Route path="crowdfunding" element={<CrowdfundingPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
