package installed

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const receiptSchemaVersion = 1

var safeReceiptName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Receipt records installation provenance outside immutable skill content.
type Receipt struct {
	SchemaVersion int       `json:"schemaVersion"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Source        string    `json:"source"`
	Digest        string    `json:"digest"`
	InstalledAt   time.Time `json:"installedAt"`
}

func WriteReceipt(skillsRoot string, receipt Receipt) error {
	if !safeReceiptName.MatchString(receipt.Name) {
		return fmt.Errorf("invalid receipt skill name %q", receipt.Name)
	}
	receipt.SchemaVersion = receiptSchemaVersion
	if receipt.InstalledAt.IsZero() {
		receipt.InstalledAt = time.Now().UTC()
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling receipt: %w", err)
	}
	data = append(data, '\n')
	dir := filepath.Join(skillsRoot, ".skillimage", "receipts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating receipt directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".receipt-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary receipt: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing receipt: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing receipt: %w", err)
	}
	path := filepath.Join(dir, receipt.Name+".json")
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("installing receipt: %w", err)
	}
	return nil
}

func readReceipts(skillsRoot string) map[string]Receipt {
	receipts := make(map[string]Receipt)
	dir := filepath.Join(skillsRoot, ".skillimage", "receipts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return receipts
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var receipt Receipt
		if json.Unmarshal(data, &receipt) != nil || receipt.SchemaVersion != receiptSchemaVersion || !safeReceiptName.MatchString(receipt.Name) {
			continue
		}
		receipts[receipt.Name] = receipt
	}
	return receipts
}
