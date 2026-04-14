package codepulse

import (
	"context"
	"errors"
	"log"
	"time"

	"go-chain/backend/internal/handlers"
	"go-chain/backend/internal/models"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

const initiatorReconcileDerived = "subgraph_reconcile"
const initiatorRPCReconcileDerived = "rpc_reconcile"

// ReconcileProposalInitiatorRolesFromSubgraph 用子图折叠出的允许集覆盖 cp_wallet_roles（全局 proposal_initiator）：
// 在集合内的设为 active=true；链上已撤销的从集合消失，对应库行 active=false。
func ReconcileProposalInitiatorRolesFromSubgraph(ctx context.Context, h *handlers.Handlers) error {
	if h == nil || h.DB == nil {
		return errors.New("reconcile: no database")
	}
	allowlist, err := queryProposalInitiatorAllowlistFromSubgraph(ctx, h)
	if err != nil {
		return err
	}
	allowed := make(map[string]struct{}, len(allowlist))
	for _, a := range allowlist {
		allowed[normalizeAddress(a)] = struct{}{}
	}

	now := time.Now().UTC()
	return h.DB.Transaction(func(tx *gorm.DB) error {
		if len(allowed) == 0 {
			return tx.Model(&models.CPWalletRole{}).
				Where("role = ? AND scope_type = ? AND active = ? AND (scope_id IS NULL OR scope_id = '')",
					"proposal_initiator", "global", true).
				Updates(map[string]any{
					"active":        false,
					"derived_from":  initiatorReconcileDerived,
					"source":        initiatorReconcileDerived,
					"updated_at":    now,
				}).Error
		}

		for a := range allowed {
			var existing models.CPWalletRole
			q := tx.Where("LOWER(wallet_address) = ? AND role = ? AND scope_type = ? AND (scope_id IS NULL OR scope_id = '')",
				a, "proposal_initiator", "global")
			err := q.First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				row := models.CPWalletRole{
					WalletAddress: a,
					Role:          "proposal_initiator",
					ScopeType:     "global",
					Active:        true,
					DerivedFrom:   initiatorReconcileDerived,
					Source:        initiatorReconcileDerived,
					CreatedAt:     now,
					UpdatedAt:     now,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			if err := tx.Model(&existing).Updates(map[string]any{
				"active":        true,
				"derived_from":  initiatorReconcileDerived,
				"source":        initiatorReconcileDerived,
				"updated_at":    now,
			}).Error; err != nil {
				return err
			}
		}

		keys := make([]string, 0, len(allowed))
		for k := range allowed {
			keys = append(keys, k)
		}
		return tx.Model(&models.CPWalletRole{}).
			Where("role = ? AND scope_type = ? AND active = ? AND (scope_id IS NULL OR scope_id = '') AND LOWER(wallet_address) NOT IN ?",
				"proposal_initiator", "global", true, keys).
			Updates(map[string]any{
				"active":        false,
				"derived_from":  initiatorReconcileDerived,
				"source":        initiatorReconcileDerived,
				"updated_at":    now,
			}).Error
	})
}

// ReconcileProposalInitiatorRolesFromRPC 在子图不可用时，对已出现在库中的地址逐个调用 isProposalInitiator 刷新 active；
// 无法发现「从未写入过库的链上新 initiator」，子图恢复后应由子图对账补齐。
func ReconcileProposalInitiatorRolesFromRPC(ctx context.Context, h *handlers.Handlers) error {
	if h == nil || h.DB == nil || h.CodePulse == nil {
		return errors.New("reconcile rpc: need database and code pulse contract")
	}
	var rows []models.CPWalletRole
	if err := h.DB.Where("role = ? AND scope_type = ? AND (scope_id IS NULL OR scope_id = '')",
		"proposal_initiator", "global").Find(&rows).Error; err != nil {
		return err
	}
	seen := make(map[string]struct{})
	var addrs []string
	for _, r := range rows {
		a := normalizeAddress(r.WalletAddress)
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		addrs = append(addrs, a)
	}
	if len(addrs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	return h.DB.Transaction(func(tx *gorm.DB) error {
		for _, a := range addrs {
			active, err := h.CodePulse.IsProposalInitiator(ctx, common.HexToAddress(a))
			if err != nil {
				return err
			}
			var existing models.CPWalletRole
			err = tx.Where("LOWER(wallet_address) = ? AND role = ? AND scope_type = ? AND (scope_id IS NULL OR scope_id = '')",
				a, "proposal_initiator", "global").First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if !active {
					continue
				}
				row := models.CPWalletRole{
					WalletAddress: a,
					Role:          "proposal_initiator",
					ScopeType:     "global",
					Active:        true,
					DerivedFrom:   initiatorRPCReconcileDerived,
					Source:        initiatorRPCReconcileDerived,
					CreatedAt:     now,
					UpdatedAt:     now,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			if err := tx.Model(&existing).Updates(map[string]any{
				"active":        active,
				"derived_from":  initiatorRPCReconcileDerived,
				"source":        initiatorRPCReconcileDerived,
				"updated_at":    now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ReconcileProposalInitiatorRoles 先尝试子图全量对账；失败且配置了合约时回退到 RPC 逐地址刷新。
func ReconcileProposalInitiatorRoles(ctx context.Context, h *handlers.Handlers) error {
	if h == nil || h.DB == nil {
		return nil
	}
	if h.SubgraphCodePulse != nil && h.SubgraphCodePulse.Configured() {
		if serr := ReconcileProposalInitiatorRolesFromSubgraph(ctx, h); serr == nil {
			return nil
		} else {
			log.Printf("code-pulse initiator reconcile: subgraph align failed: %v", serr)
		}
	}
	if h.CodePulse != nil {
		return ReconcileProposalInitiatorRolesFromRPC(ctx, h)
	}
	return errors.New("reconcile: need SUBGRAPH_CODE_PULSE_URL and/or CODE_PULSE for RPC fallback")
}

// RunProposalInitiatorReconcileLoop 定时将库中 proposal_initiator 与链上白名单对齐（子图优先，失败则 RPC）。
// every <= 0 时不应调用本函数。启动后会先立即跑一轮，再按间隔重复。
func RunProposalInitiatorReconcileLoop(ctx context.Context, h *handlers.Handlers, every time.Duration) {
	if h == nil || every <= 0 {
		return
	}
	run := func() {
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		err := ReconcileProposalInitiatorRoles(cctx, h)
		cancel()
		if err != nil {
			log.Printf("code-pulse initiator reconcile: %v", err)
		}
	}
	run()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}
