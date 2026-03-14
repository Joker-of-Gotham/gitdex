package platform

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
)

const releaseAssetLedgerVersion = 1

type ReleaseAssetLedgerEntry struct {
	Ref                    string            `json:"ref"`
	AssetName              string            `json:"asset_name,omitempty"`
	Digest                 string            `json:"digest,omitempty"`
	BytesPath              string            `json:"bytes_path,omitempty"`
	Size                   int               `json:"size,omitempty"`
	ContentType            string            `json:"content_type,omitempty"`
	SourceKind             string            `json:"source_kind,omitempty"`
	ReleaseID              string            `json:"release_id,omitempty"`
	Recoverability         string            `json:"recoverability,omitempty"`
	PartialRestoreRequired bool              `json:"partial_restore_required,omitempty"`
	Metadata               map[string]string `json:"metadata,omitempty"`
	CreatedAt              time.Time         `json:"created_at,omitempty"`
	UpdatedAt              time.Time         `json:"updated_at,omitempty"`
}

type releaseAssetLedger struct {
	Version   int                                   `json:"version"`
	UpdatedAt time.Time                             `json:"updated_at,omitempty"`
	Entries   map[string]ReleaseAssetLedgerEntry    `json:"entries,omitempty"`
}

func StoreReleaseAssetBytes(name string, data []byte, metadata map[string]string) (ReleaseAssetLedgerEntry, error) {
	dir, err := config.GlobalConfigDir()
	if err != nil {
		return ReleaseAssetLedgerEntry{}, err
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	safeName := sanitizeReleaseAssetLedgerName(name)
	if safeName == "" {
		safeName = "asset.bin"
	}
	ref := "release-asset:" + digest + ":" + safeName
	bytesDir := filepath.Join(dir, "release-assets", digest)
	if err := os.MkdirAll(bytesDir, 0o755); err != nil {
		return ReleaseAssetLedgerEntry{}, err
	}
	bytesPath := filepath.Join(bytesDir, safeName)
	if err := os.WriteFile(bytesPath, data, 0o600); err != nil {
		return ReleaseAssetLedgerEntry{}, err
	}

	now := time.Now()
	entry := ReleaseAssetLedgerEntry{
		Ref:                    ref,
		AssetName:              safeName,
		Digest:                 digest,
		BytesPath:              bytesPath,
		Size:                   len(data),
		ContentType:            strings.TrimSpace(metadata["content_type"]),
		SourceKind:             strings.TrimSpace(metadata["source_kind"]),
		ReleaseID:              strings.TrimSpace(metadata["release_id"]),
		Recoverability:         strings.TrimSpace(firstNonEmpty(metadata["recoverability"], metadata["rollback_grade"], "reversible")),
		PartialRestoreRequired: strings.EqualFold(strings.TrimSpace(metadata["partial_restore_required"]), "true"),
		Metadata:               cloneStringMap(metadata),
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	ledgerPath := filepath.Join(dir, "release-asset-ledger.json")
	ledger := loadReleaseAssetLedger(ledgerPath)
	if existing, ok := ledger.Entries[ref]; ok {
		entry.CreatedAt = existing.CreatedAt
	}
	if ledger.Entries == nil {
		ledger.Entries = map[string]ReleaseAssetLedgerEntry{}
	}
	ledger.Version = releaseAssetLedgerVersion
	ledger.UpdatedAt = now
	ledger.Entries[ref] = entry
	if err := saveReleaseAssetLedger(ledgerPath, ledger); err != nil {
		return ReleaseAssetLedgerEntry{}, err
	}
	return entry, nil
}

func ResolveReleaseAssetRef(ref string) (ReleaseAssetLedgerEntry, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ReleaseAssetLedgerEntry{}, fmt.Errorf("release asset ref is required")
	}
	dir, err := config.GlobalConfigDir()
	if err != nil {
		return ReleaseAssetLedgerEntry{}, err
	}
	ledger := loadReleaseAssetLedger(filepath.Join(dir, "release-asset-ledger.json"))
	entry, ok := ledger.Entries[ref]
	if !ok {
		return ReleaseAssetLedgerEntry{}, fmt.Errorf("release asset ref %q not found", ref)
	}
	return entry, nil
}

func loadReleaseAssetLedger(path string) releaseAssetLedger {
	data, err := os.ReadFile(path)
	if err != nil {
		return releaseAssetLedger{Version: releaseAssetLedgerVersion, Entries: map[string]ReleaseAssetLedgerEntry{}}
	}
	var ledger releaseAssetLedger
	if json.Unmarshal(data, &ledger) != nil {
		return releaseAssetLedger{Version: releaseAssetLedgerVersion, Entries: map[string]ReleaseAssetLedgerEntry{}}
	}
	if ledger.Entries == nil {
		ledger.Entries = map[string]ReleaseAssetLedgerEntry{}
	}
	if ledger.Version == 0 {
		ledger.Version = releaseAssetLedgerVersion
	}
	return ledger
}

func saveReleaseAssetLedger(path string, ledger releaseAssetLedger) error {
	raw, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func sanitizeReleaseAssetLedgerName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	replacer := strings.NewReplacer("<", "-", ">", "-", ":", "-", "\"", "-", "/", "-", "\\", "-", "|", "-", "?", "-", "*", "-")
	return strings.TrimSpace(replacer.Replace(name))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
