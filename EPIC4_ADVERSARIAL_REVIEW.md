# Epic 4 Adversarial Code Review Report

**Scope:** Go CLI collaboration package (objects, mutations, triage, context, release) and CLI commands  
**Review Layers:** Blind Hunter | Edge Case Hunter | Acceptance Auditor

---

## HIGH Severity

### H1. Triage/Summary use fresh store — triage always empty
**File:** `internal/cli/command/collab.go:57,107`  
**Layer:** Blind Hunter

**Issue:** `newCollabTriageCommand` and `newCollabSummaryCommand` instantiate `collaboration.NewMemoryObjectStore()` on each run instead of using the shared `collabObjectStore`. As a result, triage and summary always operate on an empty store and will never show objects created via `collab create` or listed via `collab list`.

**Fix:**
```go
// In newCollabTriageCommand (line 57) and newCollabSummaryCommand (line 107)
// Change:
store := collaboration.NewMemoryObjectStore()
// To:
store := collabObjectStore
```

---

### H2. Missing defensive copies of Assignees/Labels in ObjectStore
**File:** `internal/collaboration/objects.go:103-106,116-118,125-134,138-149`  
**Layer:** Acceptance Auditor / Blind Hunter

**Issue:** `SaveObject` does `dup := *obj` (shallow copy); `GetObject`, `ListObjects`, and `GetByRepoAndNumber` return `out := *obj` without copying `Assignees` and `Labels`. Callers receive slices that reference internal store data; mutations will corrupt the store and create data races under concurrent use.

**Fix:**
```go
// In SaveObject, after dup := *obj:
dup.Assignees = append([]string(nil), obj.Assignees...)
dup.Labels = append([]string(nil), obj.Labels...)

// In GetObject, ListObjects, GetByRepoAndNumber, when building the returned copy:
out.Assignees = append([]string(nil), obj.Assignees...)
out.Labels = append([]string(nil), obj.Labels...)
```

---

### H3. ObjectRef format mismatch — PR refs cannot be used with show/comment/close
**File:** `internal/collaboration/triage.go:183-189`, `internal/cli/command/collab.go:431-446`  
**Layer:** Edge Case Hunter / Blind Hunter

**Issue:** Triage uses `ObjectRef(obj)`, which for pull requests returns `owner/repo#pr/42`. `parseObjectRef` expects `owner/repo#number` and treats `#pr/42` as an invalid number, so commands like `show`, `comment`, `close`, `reopen` cannot use triage output for PRs.

**Fix:** Either:
1. Normalize refs to `owner/repo#number` in triage (align with `CollaborationObject.ObjectRef()` and `repoNumberKey`), or  
2. Extend `parseObjectRef` to accept `#pr/N` and map it to the same owner/repo/number.

---

### H4. Nil request causes panic in MutationEngine.Execute
**File:** `internal/collaboration/mutations.go:60-61`  
**Layer:** Blind Hunter / Edge Case Hunter

**Issue:** If `req` is nil, `req.MutationType` panics.

**Fix:**
```go
func (e *SimulatedMutationEngine) Execute(ctx context.Context, req *MutationRequest) (*MutationResult, error) {
	if req == nil {
		return nil, fmt.Errorf("mutation request cannot be nil")
	}
	switch req.MutationType {
```

---

## MEDIUM Severity

### M1. MutationEngine mutates store object and exposes internal pointer
**File:** `internal/collaboration/mutations.go:126-136,138-144` (executeComment, executeClose, etc.)  
**Layer:** Blind Hunter / Acceptance Auditor

**Issue:** Mutation handlers return `Object: obj` where `obj` is the pointer obtained from `GetByRepoAndNumber`, then mutated. The store returns a defensive copy for the struct, but `MutationResult.Object` can still share internal data through slice fields. Also, mutating the copy and then saving is correct, but returning the same pointer in `MutationResult` allows callers to mutate the object and affect future reads until slices are properly copied.

**Fix:** After `SaveObject`, either return `nil` for `Object` or construct a fresh defensive copy before populating `MutationResult.Object`.

---

### M2. SearchQuery in ObjectFilter is never applied
**File:** `internal/collaboration/objects.go:161-214`  
**Layer:** Blind Hunter / Edge Case Hunter

**Issue:** `ObjectFilter.SearchQuery` exists but `matchFilter` never uses it. Filtering by search query has no effect.

**Fix:**
```go
if filter.SearchQuery != "" {
	if !strings.Contains(strings.ToLower(obj.Title+" "+obj.Body), strings.ToLower(filter.SearchQuery)) {
		return false
	}
}
```
(Adjust semantics if search should match other fields.)

---

### M3. executeCreate does not validate RepoOwner/RepoName
**File:** `internal/collaboration/mutations.go:83-95`  
**Layer:** Edge Case Hunter

**Issue:** Empty `req.RepoOwner` or `req.RepoName` produces keys like `/#1` and URLs like `https://github.com//issues/1`, which are invalid.

**Fix:**
```go
if req.RepoOwner == "" || req.RepoName == "" {
	return &MutationResult{Request: *req, Success: false, Message: "repo owner and name are required"}, nil
}
```

---

### M4. Mutations share slice references with request
**File:** `internal/collaboration/mutations.go:107-108,204-207`  
**Layer:** Blind Hunter / Acceptance Auditor

**Issue:** `obj.Assignees = req.Assignees` and `obj.Labels = req.Labels` share backing arrays. Later mutations of `req` affect stored data.

**Fix:**
```go
Assignees: append([]string(nil), req.Assignees...),
Labels:    append([]string(nil), req.Labels...),
// And in executeUpdate:
if len(req.Labels) > 0 {
	obj.Labels = append([]string(nil), req.Labels...)
}
if len(req.Assignees) > 0 {
	obj.Assignees = append([]string(nil), req.Assignees...)
}
```

---

### M5. SaveContext can orphan contexts when PrimaryObjectRef is reused
**File:** `internal/collaboration/context.go:80-103`  
**Layer:** Edge Case Hunter

**Issue:** `byObjRef` maps one context per `PrimaryObjectRef`. Saving a new context with the same `PrimaryObjectRef` but different `ContextID` overwrites `byObjRef` but leaves the old context in `byID`. `ListContexts` will still return the orphaned entry.

**Fix:** Before overwriting `byObjRef`, remove the old context’s `ContextID` from `byID`:
```go
if old, ok := s.byObjRef[dup.PrimaryObjectRef]; ok && old.ContextID != dup.ContextID {
	delete(s.byID, old.ContextID)
}
```

---

### M6. Release commands ignore repo root
**File:** `internal/cli/command/release.go:40,75`  
**Layer:** Acceptance Auditor

**Issue:** Release commands pass `parseRepoFlag(repoFlag, "")` instead of `app.RepoRoot`, so `--repo` is always required and the current git repo is never inferred, unlike collab commands.

**Fix:**
```go
owner, repoName := parseRepoFlag(repoFlag, app.RepoRoot)
```

---

## LOW Severity

### L1. SaveObject mutates caller's object
**File:** `internal/collaboration/objects.go:86-92`  
**Layer:** Blind Hunter

**Issue:** When `obj.ObjectID == ""` or `obj.CreatedAt.IsZero()`, `SaveObject` mutates the input. Callers may not expect this.

**Fix:** Either document this behavior or clone the object at the start and work only on the clone.

---

### L2. Summarize uses first object for RepoOwner/RepoName
**File:** `internal/collaboration/triage.go:137-141`  
**Layer:** Edge Case Hunter

**Issue:** If `objects` contain multiple repos, summary reports only `objects[0]`’s owner/repo, which can be misleading.

**Fix:** Document or change behavior (e.g., support multi-repo summaries or require a single repo).

---

### L3. Link command does not validate ref format
**File:** `internal/cli/command/collab.go:153-157`  
**Layer:** Edge Case Hunter

**Issue:** `args[0]` and `args[1]` are used as refs without validating `owner/repo#number` (or equivalent). Invalid refs are stored.

**Fix:** Validate with `parseObjectRef` or a lighter check before creating links.

---

### L4. Duplicate links allowed
**File:** `internal/cli/command/collab.go:165`  
**Layer:** Edge Case Hunter

**Issue:** `tc.LinkedObjects = append(tc.LinkedObjects, *link)` allows identical links (same source, target, type) to be added repeatedly.

**Fix:** Optionally check for duplicates before appending.

---

### L5. GetByObjectRef error ignored in link command
**File:** `internal/cli/command/collab.go:158`  
**Layer:** Blind Hunter

**Issue:** `tc, _ := collabContextStore.GetByObjectRef(...)` ignores the error. That’s acceptable when treating “not found” as “create new,” but any other error is silently dropped.

**Fix:** If errors other than “not found” should be surfaced:
```go
tc, err := collabContextStore.GetByObjectRef(context.Background(), args[0])
if err != nil && !errors.Is(err, ErrNotFound) {
	return fmt.Errorf("get context: %w", err)
}
```

---

## Summary by Severity

| Severity | Count |
|----------|-------|
| HIGH     | 4     |
| MEDIUM   | 6     |
| LOW      | 5     |

## Summary by Criteria

| Criteria                  | Status |
|---------------------------|--------|
| Thread-safe stores        | ✅     |
| Defensive copies          | ❌ ObjectStore slices |
| CLI args validated        | ✅ (except link refs) |
| Structured output         | ✅     |
| Error messages            | ✅ Mostly |
| JSON/YAML tags            | ✅     |
