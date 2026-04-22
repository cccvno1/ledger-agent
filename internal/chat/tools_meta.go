package chat

import "time"

// This file declares Capability, AllowedPhases, and Timeout for every tool
// in tools.go so they satisfy the ManagedTool interface. Keeping the metadata
// in one place (rather than scattered next to each tool's Info) makes the
// privilege matrix auditable at a glance.
//
// Phase A (current): metadata is recorded for observability only; the
// Registry does not yet enforce phase restrictions. Phase B will add a Policy
// layer that consults AllowedPhases() before dispatch.

// --- search_customer ---
func (*searchCustomerTool) Capability() Capability { return CapRead }
func (*searchCustomerTool) AllowedPhases() []Phase { return nil }
func (*searchCustomerTool) Timeout() time.Duration { return 0 }

// --- list_customers ---
func (*listCustomersTool) Capability() Capability { return CapRead }
func (*listCustomersTool) AllowedPhases() []Phase { return nil }
func (*listCustomersTool) Timeout() time.Duration { return 0 }

// --- add_to_draft ---
func (*addToDraftTool) Capability() Capability { return CapMutate }
func (*addToDraftTool) AllowedPhases() []Phase {
	return []Phase{PhaseIdle, PhaseDrafting, PhaseCommitted}
}
func (*addToDraftTool) Timeout() time.Duration { return 0 }

// --- update_draft_item ---
func (*updateDraftItemTool) Capability() Capability { return CapMutate }
func (*updateDraftItemTool) AllowedPhases() []Phase { return []Phase{PhaseDrafting} }
func (*updateDraftItemTool) Timeout() time.Duration { return 0 }

// --- remove_draft_item ---
func (*removeDraftItemTool) Capability() Capability { return CapMutate }
func (*removeDraftItemTool) AllowedPhases() []Phase { return []Phase{PhaseDrafting} }
func (*removeDraftItemTool) Timeout() time.Duration { return 0 }

// --- clear_draft ---
func (*clearDraftTool) Capability() Capability { return CapMutate }
func (*clearDraftTool) AllowedPhases() []Phase { return []Phase{PhaseDrafting} }
func (*clearDraftTool) Timeout() time.Duration { return 0 }

// --- confirm_draft ---
func (*confirmDraftTool) Capability() Capability { return CapCommit }
func (*confirmDraftTool) AllowedPhases() []Phase { return []Phase{PhaseDrafting, PhasePendingCommit} }
func (*confirmDraftTool) Timeout() time.Duration { return 30 * time.Second }

// --- query_entries ---
func (*queryEntriesTool) Capability() Capability { return CapRead }
func (*queryEntriesTool) AllowedPhases() []Phase { return nil }
func (*queryEntriesTool) Timeout() time.Duration { return 0 }

// --- update_entry ---
func (*updateEntryTool) Capability() Capability { return CapMutate }
func (*updateEntryTool) AllowedPhases() []Phase { return nil } // any phase, post-commit edits allowed
func (*updateEntryTool) Timeout() time.Duration { return 0 }

// --- propose_delete_entry ---
func (*proposeDeleteEntryTool) Capability() Capability { return CapMutate }
func (*proposeDeleteEntryTool) AllowedPhases() []Phase {
	return []Phase{PhaseIdle, PhaseCommitted, PhasePendingDestructive}
}
func (*proposeDeleteEntryTool) Timeout() time.Duration { return 0 }

// --- propose_settle_account ---
func (*proposeSettleAccountTool) Capability() Capability { return CapMutate }
func (*proposeSettleAccountTool) AllowedPhases() []Phase {
	return []Phase{PhaseIdle, PhaseCommitted, PhasePendingDestructive}
}
func (*proposeSettleAccountTool) Timeout() time.Duration { return 0 }

// --- commit_operation ---
// Only allowed in PendingDestructive: structurally enforces that the agent
// must have proposed an operation before committing it.
func (*commitOperationTool) Capability() Capability { return CapDestructive }
func (*commitOperationTool) AllowedPhases() []Phase { return []Phase{PhasePendingDestructive} }
func (*commitOperationTool) Timeout() time.Duration { return 30 * time.Second }

// --- calculate_summary ---
func (*calculateSummaryTool) Capability() Capability { return CapRead }
func (*calculateSummaryTool) AllowedPhases() []Phase { return nil }
func (*calculateSummaryTool) Timeout() time.Duration { return 0 }

// --- record_payment ---
func (*recordPaymentTool) Capability() Capability { return CapCommit }
func (*recordPaymentTool) AllowedPhases() []Phase { return nil }
func (*recordPaymentTool) Timeout() time.Duration { return 0 }

// --- list_products ---
func (*listProductsTool) Capability() Capability { return CapRead }
func (*listProductsTool) AllowedPhases() []Phase { return nil }
func (*listProductsTool) Timeout() time.Duration { return 0 }

// --- list_payments ---
func (*listPaymentsTool) Capability() Capability { return CapRead }
func (*listPaymentsTool) AllowedPhases() []Phase { return nil }
func (*listPaymentsTool) Timeout() time.Duration { return 0 }

// --- add_customer_alias ---
func (*addCustomerAliasTool) Capability() Capability { return CapMutate }
func (*addCustomerAliasTool) AllowedPhases() []Phase { return nil }
func (*addCustomerAliasTool) Timeout() time.Duration { return 0 }

// --- add_product_alias ---
func (*addProductAliasTool) Capability() Capability { return CapMutate }
func (*addProductAliasTool) AllowedPhases() []Phase { return nil }
func (*addProductAliasTool) Timeout() time.Duration { return 0 }
