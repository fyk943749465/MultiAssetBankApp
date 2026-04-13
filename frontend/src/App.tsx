import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AppLayout } from "./layouts/AppLayout";
import { BankPage } from "./pages/BankPage";
import { CrowdfundingPage } from "./pages/CrowdfundingPage";
import { CampaignDetailPage } from "./pages/crowdfunding/CampaignDetailPage";
import { CrowdfundingAdminPage } from "./pages/crowdfunding/CrowdfundingAdminPage";
import { CrowdfundingExplorePage } from "./pages/crowdfunding/CrowdfundingExplorePage";
import { CrowdfundingHomePage } from "./pages/crowdfunding/CrowdfundingHomePage";
import { CrowdfundingNewProposalPage } from "./pages/crowdfunding/CrowdfundingNewProposalPage";
import { CrowdfundingWorkspacePage } from "./pages/crowdfunding/CrowdfundingWorkspacePage";
import { ProposalDetailPage } from "./pages/crowdfunding/ProposalDetailPage";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<Navigate to="/bank" replace />} />
          <Route path="bank" element={<BankPage />} />
          <Route path="crowdfunding" element={<CrowdfundingPage />}>
            <Route index element={<CrowdfundingHomePage />} />
            <Route path="explore" element={<CrowdfundingExplorePage />} />
            <Route path="me" element={<CrowdfundingWorkspacePage />} />
            <Route path="me/proposals/new" element={<CrowdfundingNewProposalPage />} />
            <Route path="admin" element={<CrowdfundingAdminPage />} />
            <Route path="proposals/:proposalId" element={<ProposalDetailPage />} />
            <Route path="campaigns/:campaignId" element={<CampaignDetailPage />} />
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
