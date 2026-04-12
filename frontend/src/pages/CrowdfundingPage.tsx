import { sectionTitle, surfaceMuted } from "../ui/styles";

/** 众筹合约业务页：与银行/后端模块分离，后续在此挂载众筹相关组件与路由子段。 */
export function CrowdfundingPage() {
  return (
    <div className="space-y-6">
      <p className="text-center text-sm leading-relaxed text-slate-400">
        本页为众筹合约专用入口，与「银行与后端」无数据与 UI 耦合；仅共用顶栏钱包连接。
      </p>

      <div className={surfaceMuted}>
        <h2 className={sectionTitle}>众筹合约</h2>
        <p className="text-sm text-slate-400">
          功能开发中。可在此逐步加入列表、创建众筹、参与、领取等模块。
        </p>
      </div>
    </div>
  );
}
