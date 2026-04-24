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
import { NftPage } from "./pages/nft/NftPage";
import { NftHomePage } from "./pages/nft/NftHomePage";
import { NftCollectionDetailPage } from "./pages/nft/NftCollectionDetailPage";
import { NftCollectionMintPage } from "./pages/nft/NftCollectionMintPage";
import { NftCreateCollectionPage } from "./pages/nft/NftCreateCollectionPage";
import { NftMyHoldingsPage } from "./pages/nft/NftMyHoldingsPage";
import { NftMarketPage } from "./pages/nft/NftMarketPage";
import { NftMarketHistoryPage } from "./pages/nft/NftMarketHistoryPage";
import { LendingPage } from "./pages/lending/LendingPage";
import { LendingHomePage } from "./pages/lending/LendingHomePage";
import { LendingPoolPage } from "./pages/lending/LendingPoolPage";

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
          <Route path="nft" element={<NftPage />}>
            <Route index element={<NftHomePage />} />
            <Route path="me" element={<NftMyHoldingsPage />} />
            <Route path="market/history" element={<NftMarketHistoryPage />} />
            <Route path="market" element={<NftMarketPage />} />
            <Route path="create" element={<NftCreateCollectionPage />} />
            <Route path="collections/:contractAddress/mint" element={<NftCollectionMintPage />} />
            <Route path="collections/:collectionId" element={<NftCollectionDetailPage />} />
          </Route>
          <Route path="lending" element={<LendingPage />}>
            <Route index element={<LendingHomePage />} />
            <Route path="pool" element={<LendingPoolPage />} />
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
