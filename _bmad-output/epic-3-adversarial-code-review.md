# Epic 3 Adversarial Code Review Report

**Epic:** Go CLI (Identity, Policy, Audit, Emergency)  
**Review Date:** 2025-03-19  
**Review Layers:** Blind Hunter, Edge Case Hunter, Acceptance Auditor

---

## Executive Summary

Three parallel review layers were run on all Epic 3 code. **3 HIGH** severity issues were identified and **fixed**. Several **MEDIUM** and **LOW** findings remain documented below for follow-up.

---

## HIGH Severity (Fixed)

### H1: Audit ledger returns internal pointers – data corruption risk
- **File:** `internal/audit/ledger.go`
- **Layer:** Blind Hunter, Acceptance Auditor
- **Issue:** `Query`, `GetByCorrelation`, `GetByTask`, and `GetByEntryID` returned pointers to internal entries. Callers could mutate returned entries and corrupt the ledger state.
- **Fix applied:** Each method now returns defensive copies. `Query` builds `out` with `cp := *e; cp.EvidenceRefs = make(...); copy(...); out = append(out, &cp)`. `GetByEntryID` returns `&cp` with copied `EvidenceRefs`.

### H2: Policy bundle shallow copies – shared mutable state
- **File:** `internal/policy/bundle.go`
- **Layer:** Acceptance Auditor
- **Issue:** `SaveBundle`, `GetBundle`, and `ListBundles` used shallow copies. `RiskThresholds` map, `DataHandlingRules` slice, and nested slices inside `CapabilityGrant` (Capabilities, Conditions) and `ApprovalRule` (RequiredApprovers) were shared. Callers could mutate returned data and affect stored bundles.
- **Fix applied:** Added `deepCopyBundle(b *PolicyBundle)` that deep-copies all slices and maps. All store methods now use `deepCopyBundle` when storing and when returning data.

### H3: Audit log `--limit` negative value causes panic
- **File:** `internal/cli/command/audit.go:52`
- **Layer:** Edge Case Hunter
- **Issue:** `entries[len(entries)-limit:]` with `limit < 0` (e.g. `--limit -1`) could produce invalid slice indices and panic.
- **Fix applied:** Added validation: `if limit < 0 { limit = 0 }` before the slice operation.

---

## MEDIUM Severity (Documented)

### M1: Store mutations of caller input
- **Files:** `internal/identity/app_identity.go:82-86`, `internal/policy/bundle.go:129-136`
- **Layer:** Blind Hunter
- **Issue:** `SaveIdentity` and `SaveBundle` mutate the input struct when `IdentityID`/`BundleID`, `Version`, or `CreatedAt` are empty. Callers may not expect side effects on their objects.
- **Suggestion:** Prefer creating a local copy before mutation, or document that input may be mutated. Current usage in CLI relies on this for `SetCurrentIdentity(ident.IdentityID)` after save, so it works but is surprising.

### M2: Emergency `Execute` returns `nil` error for unknown action
- **File:** `internal/emergency/controls.go:82`
- **Layer:** Blind Hunter
- **Issue:** When `request.Action` is unknown, `Execute` returns `(result, nil)` with `Success: false`. Callers that only check `err != nil` may miss failure.
- **Suggestion:** Consider `return result, fmt.Errorf("unknown control action: %s", request.Action)` to align with interface semantics.

### M3: Identity `GetCurrentIdentity` TOCTOU semantics
- **File:** `internal/identity/app_identity.go:139-148`
- **Layer:** Edge Case Hunter
- **Issue:** Between reading `currentID` and calling `GetIdentity(currentID)`, another goroutine could call `SetCurrentIdentity` to a different ID. The returned identity may not match the “current” at the moment of use. Low impact since there is no delete, but semantically stale.
- **Suggestion:** Hold read lock for the entire operation if strict consistency is required; otherwise document as best-effort.

---

## LOW Severity (Documented)

### L1: `AuditFilter` missing JSON/YAML tags
- **File:** `internal/audit/ledger.go:39-45`
- **Layer:** Acceptance Auditor
- **Issue:** Exported `AuditFilter` has no JSON/YAML tags. Likely internal-only; add tags if ever serialized.

### L2: `truncate` can panic if `max <= 0`
- **File:** `internal/cli/command/audit.go:156-162`
- **Layer:** Edge Case Hunter
- **Issue:** `s[:max-1]` with `max <= 0` would panic. Currently only called with constants 12 and 15.
- **Suggestion:** Add guard: `if max <= 0 { return s }` for robustness.

### L3: Global `identityStore` and `policyBundleStore` not injectable
- **Files:** `internal/cli/command/identity.go:16`, `internal/cli/command/policy.go:16`
- **Layer:** Acceptance Auditor
- **Issue:** Package-level store vars are not injectable for tests (unlike `auditLedger` which has `SetAuditLedgerForTest`).
- **Suggestion:** Add `SetIdentityStoreForTest` and `SetPolicyBundleStoreForTest` for integration tests.

### L4: Empty `bundleID` in policy store
- **File:** `internal/policy/bundle.go:186-188`
- **Layer:** Edge Case Hunter
- **Issue:** `SetActiveBundle("")` looks up `s.bundles[""]`, which typically fails with “bundle \"\" not found”. Behavior is correct; no change needed.

---

## Verification

- Thread-safety: All stores use `sync.RWMutex` correctly; reads use `RLock`, writes use `Lock`.
- Defensive copies: Identity, policy, and audit return full defensive copies (including nested slices/maps).
- CLI validation: `identity register` requires `--app-id` and `--installation-id` for `github_app`; `policy create` uses `MarkFlagRequired("name")`; `audit show` and `audit trace` use `ExactArgs(1)`; emergency commands use `ExactArgs(1)` or `NoArgs`.
- JSON/YAML: All exported types used for output have proper tags.
- Error messages: User-facing errors are descriptive (e.g. “identity %q not found”, “--name is required”).

---

## Files Reviewed

| File | Story | Status |
|------|------|--------|
| `internal/identity/app_identity.go` | 3.1 | OK (defensive copies for ScopeGrants) |
| `internal/cli/command/identity.go` | 3.1 | OK |
| `internal/policy/bundle.go` | 3.2 | OK (deepCopyBundle) |
| `internal/cli/command/policy.go` | 3.2 | OK |
| `internal/audit/ledger.go` | 3.3 | OK (defensive copies) |
| `internal/cli/command/audit.go` | 3.3 | OK (limit validation) |
| `internal/emergency/controls.go` | 3.4 | OK |
| `internal/cli/command/emergency.go` | 3.4 | OK |

---

## Conclusion

All HIGH severity issues have been fixed. The codebase meets the acceptance criteria for thread-safety, defensive copies, CLI validation, structured output, and exported type tags. MEDIUM and LOW findings are documented for future improvements.
