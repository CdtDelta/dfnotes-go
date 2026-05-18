package export

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"dfnotes-go/internal/ioc"
	"dfnotes-go/internal/models"
	"dfnotes-go/internal/services"
)

const appVersion = "0.4.0"

var (
	reSanitizeTS      = regexp.MustCompile(`[^a-zA-Z0-9]`)
	reSanitizeDirName = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
)

type ProgressFunc func(stage string, percent int)

type ExportRequest struct {
	CaseID          string
	ArchivePassword string
	DBPath          string
	ArchivePath     string // absolute path chosen by the user
	ExaminerName    string
	ExaminerPubKey  []byte
}

type caseMetadata struct {
	CaseID              string `json:"case_id"`
	CaseNumber          string `json:"case_number"`
	Title               string `json:"title"`
	ExaminerName        string `json:"examiner_name"`
	Organization        string `json:"organization"`
	ClassificationLevel string `json:"classification_level"`
	TicketNumber        string `json:"ticket_number"`
	Description         string `json:"description"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

type blockRecord struct {
	BlockID         string `json:"block_id"`
	Sequence        int    `json:"sequence"`
	EvidenceItemID  any    `json:"evidence_item_id"`
	CommittedAt     string `json:"committed_at"`
	ContentHash     string `json:"content_hash"`
	PrevBlockHash   string `json:"previous_block_hash"`
	Signature       string `json:"signature"`
	ChainValid      bool   `json:"chain_valid"`
}

type chainVerification struct {
	CaseID             string        `json:"case_id"`
	CaseNumber         string        `json:"case_number"`
	ExportedAt         string        `json:"exported_at"`
	ExaminerPublicKey  string        `json:"examiner_public_key"`
	Blocks             []blockRecord `json:"blocks"`
	ChainIntact        bool          `json:"chain_intact"`
	TotalBlocks        int           `json:"total_blocks"`
}

type iocExport struct {
	IOCID          string `json:"ioc_id"`
	Type           string `json:"type"`
	ValueRaw       string `json:"value_raw"`
	ValueDefanged  string `json:"value_defanged"`
	Status         string `json:"status"`
	DetectionMethod string `json:"detection_method"`
	BlockID        string `json:"block_id"`
	EvidenceItemID string `json:"evidence_item_id,omitempty"`
	Notes          string `json:"notes,omitempty"`
	CreatedAt      string `json:"created_at"`
	ConfirmedAt    string `json:"confirmed_at,omitempty"`
}

// ExportCase collects all case data, builds a directory tree, archives it with
// 7z AES-256 encryption (shelling out to the 7z CLI), and returns the archive path.
func ExportCase(
	ctx context.Context,
	req ExportRequest,
	caseData *services.CaseResponse,
	evidenceItems []services.EvidenceResponse,
	masterBlocks []services.NoteBlockResponse,
	evidenceBlockMap map[string][]services.NoteBlockResponse,
	rawBlocks []models.NoteBlock,
	iocEntries []ioc.IOCEntry,
	timelineEntries []models.TimelineEntry,
	progress ProgressFunc,
) (string, error) {
	if _, err := exec.LookPath("7z"); err != nil {
		return "", fmt.Errorf("7z binary not found -- install p7zip-full to enable export")
	}

	progress("preparing export directory", 5)

	tmpDir, err := os.MkdirTemp("", "dfnotes-export-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// --- case_metadata.json ---
	progress("writing case metadata", 10)
	meta := caseMetadata{
		CaseID:              caseData.CaseID,
		CaseNumber:          caseData.CaseNumber,
		Title:               caseData.Title,
		ExaminerName:        req.ExaminerName,
		Organization:        caseData.Organization,
		ClassificationLevel: caseData.Classification,
		TicketNumber:        caseData.TicketNumber,
		Description:         caseData.Description,
		CreatedAt:           caseData.CreatedAt,
		UpdatedAt:           caseData.UpdatedAt,
	}
	if err := writeJSON(filepath.Join(tmpDir, "case_metadata.json"), meta); err != nil {
		return "", err
	}

	// --- database copy ---
	progress("copying database", 15)
	dbDest := filepath.Join(tmpDir, caseData.CaseNumber+".db")
	if err := copyFile(req.DBPath, dbDest); err != nil {
		return "", fmt.Errorf("copy database: %w", err)
	}

	// --- master notes ---
	progress("writing note blocks", 25)
	masterDir := filepath.Join(tmpDir, "master_notes")
	if err := os.MkdirAll(masterDir, 0o700); err != nil {
		return "", err
	}
	for i, block := range masterBlocks {
		filename := fmt.Sprintf("block_%03d_%s.md", i+1, sanitizeTS(block.CreatedAt))
		content := formatBlockFile(block, "master_notes", i+1)
		if err := os.WriteFile(filepath.Join(masterDir, filename), []byte(content), 0o600); err != nil {
			return "", fmt.Errorf("write block %s: %w", block.BlockID, err)
		}
	}

	// --- evidence items ---
	progress("writing evidence items", 40)
	evidDir := filepath.Join(tmpDir, "evidence")
	if err := os.MkdirAll(evidDir, 0o700); err != nil {
		return "", err
	}
	for _, item := range evidenceItems {
		dirName := sanitizeDirName(item.EvidenceItemID + "_" + item.Name)
		itemDir := filepath.Join(evidDir, dirName)
		if err := os.MkdirAll(itemDir, 0o700); err != nil {
			return "", err
		}

		// metadata.json
		evMeta := buildEvidenceMeta(item)
		if err := writeJSON(filepath.Join(itemDir, "metadata.json"), evMeta); err != nil {
			return "", err
		}

		// note blocks
		blocks := evidenceBlockMap[item.EvidenceItemID]
		for i, block := range blocks {
			filename := fmt.Sprintf("block_%03d_%s.md", i+1, sanitizeTS(block.CreatedAt))
			content := formatBlockFile(block, item.EvidenceItemID, i+1)
			if err := os.WriteFile(filepath.Join(itemDir, filename), []byte(content), 0o600); err != nil {
				return "", fmt.Errorf("write evidence block %s: %w", block.BlockID, err)
			}
		}
	}

	// --- ioc_summary.json ---
	progress("writing IOC summary", 55)
	iocExports := make([]iocExport, 0, len(iocEntries))
	for _, e := range iocEntries {
		iocExports = append(iocExports, iocExport{
			IOCID:          e.IOCID,
			Type:           string(e.Type),
			ValueRaw:       e.Value,
			ValueDefanged:  defangGo(e.Value, string(e.Type)),
			Status:         string(e.Status),
			DetectionMethod: e.DetectionMethod,
			BlockID:        e.BlockID,
			EvidenceItemID: func() string { if e.EvidenceItemID != nil { return *e.EvidenceItemID }; return "" }(),
			Notes:          func() string { if e.Notes != nil { return *e.Notes }; return "" }(),
			CreatedAt:      e.CreatedAt,
			ConfirmedAt:    func() string { if e.ConfirmedAt != nil { return *e.ConfirmedAt }; return "" }(),
		})
	}
	if err := writeJSON(filepath.Join(tmpDir, "ioc_summary.json"), iocExports); err != nil {
		return "", err
	}

	// --- timeline.json ---
	progress("writing timeline", 65)
	if err := writeJSON(filepath.Join(tmpDir, "timeline.json"), timelineEntries); err != nil {
		return "", err
	}

	// --- chain_verification.json ---
	progress("verifying chain", 75)
	cv := buildChainVerification(caseData, rawBlocks, req.ExaminerPubKey)
	if err := writeJSON(filepath.Join(tmpDir, "chain_verification.json"), cv); err != nil {
		return "", err
	}

	// --- README.txt ---
	progress("writing README", 80)
	readme := buildREADME(caseData, req.ExaminerName)
	if err := os.WriteFile(filepath.Join(tmpDir, "README.txt"), []byte(readme), 0o600); err != nil {
		return "", fmt.Errorf("write README: %w", err)
	}

	// --- create archive ---
	progress("creating encrypted archive", 85)
	archivePath := req.ArchivePath

	if err := create7z(archivePath, tmpDir, req.ArchivePassword); err != nil {
		os.Remove(archivePath)
		return "", err
	}

	progress("done", 100)
	return archivePath, nil
}

func create7z(archivePath, srcDir, password string) error {
	// Run 7z with srcDir as the working directory so that "." refers to its
	// contents — this adds files at the archive root with no srcDir prefix.
	// -mhe=on encrypts file headers too.
	// -mx=0 stores without compression (fast; forensic archives are unambiguous).
	cmd := exec.Command("7z", "a",
		"-p"+password,
		"-mhe=on",
		"-mx=0",
		"-t7z",
		archivePath,
		".",
	)
	cmd.Dir = srcDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("7z failed: %w -- %s", err, stderr.String())
	}
	return nil
}

func formatBlockFile(block services.NoteBlockResponse, evidenceItemID string, seq int) string {
	return fmt.Sprintf(`<!--
dfnotes-go Block Record
block_id:            %s
case_id:             %s
evidence_item_id:    %s
committed_at:        %s
content_hash:        %s
previous_block_hash: %s
examiner:            %s
-->

%s
`, block.BlockID, block.CaseID, evidenceItemID, block.CreatedAt,
		block.ContentHash, block.PrevHash, block.AuthorID, block.Content)
}

func buildChainVerification(caseData *services.CaseResponse, rawBlocks []models.NoteBlock, pubKey []byte) chainVerification {
	pubKeyHex := hex.EncodeToString(pubKey)
	var records []blockRecord
	allValid := true
	for i, b := range rawBlocks {
		prevHash := b.PrevHash
		// Chain is intact if each block's prev_hash matches the previous block's content_hash.
		valid := true
		if i > 0 {
			valid = rawBlocks[i-1].ContentHash == prevHash
		}
		if !valid {
			allValid = false
		}
		var evidID any
		if b.EvidenceItemID != nil {
			evidID = *b.EvidenceItemID
		}
		records = append(records, blockRecord{
			BlockID:        b.BlockID,
			Sequence:       i + 1,
			EvidenceItemID: evidID,
			CommittedAt:    b.CreatedAt.UTC().Format(time.RFC3339),
			ContentHash:    b.ContentHash,
			PrevBlockHash:  prevHash,
			Signature:      base64.StdEncoding.EncodeToString(b.Signature),
			ChainValid:     valid,
		})
	}
	return chainVerification{
		CaseID:            caseData.CaseID,
		CaseNumber:        caseData.CaseNumber,
		ExportedAt:        time.Now().UTC().Format(time.RFC3339),
		ExaminerPublicKey: pubKeyHex,
		Blocks:            records,
		ChainIntact:       allValid,
		TotalBlocks:       len(rawBlocks),
	}
}

type evidenceMeta struct {
	ItemID          string                          `json:"item_id"`
	Name            string                          `json:"name"`
	Description     string                          `json:"description"`
	Type            string                          `json:"type"`
	AcquisitionHash string                          `json:"acquisition_hash"`
	Status          string                          `json:"status"`
	CreatedAt       string                          `json:"created_at"`
	ChainOfCustody  []services.CustodyEntryResponse `json:"chain_of_custody"`
}

func buildEvidenceMeta(item services.EvidenceResponse) evidenceMeta {
	return evidenceMeta{
		ItemID:          item.EvidenceItemID,
		Name:            item.Name,
		Description:     item.Description,
		Type:            item.EvidenceType,
		AcquisitionHash: item.ContentHash,
		Status:          item.Status,
		CreatedAt:       item.CreatedAt,
		ChainOfCustody:  item.CustodyLog,
	}
}

func buildREADME(caseData *services.CaseResponse, examiner string) string {
	return fmt.Sprintf(`dfnotes-go Case Export
======================
Case Number: %s
Exported At: %s
Exported By: %s
dfnotes-go Version: %s

CONTENTS
--------
%s.db              - Encrypted SQLite database. Requires dfnotes-go to decrypt.
case_metadata.json  - Case metadata in plaintext JSON.
master_notes/       - Committed note blocks from Master Notes tab (markdown).
evidence/           - Per-evidence-item metadata and note blocks.
ioc_summary.json    - All IOC entries with status and source references.
timeline.json       - All timeline entries.
chain_verification.json - Full hash chain data for independent verification.

VERIFICATION
------------
Each markdown block file contains a header with the block's content hash,
previous block hash, and examiner ID. The chain_verification.json file
contains the complete chain in machine-readable form.

To independently verify the chain integrity:
1. Open chain_verification.json.
2. For each block, verify content_hash matches the block's plaintext content.
3. Verify each block's previous_block_hash matches the prior block's content_hash.
4. Use the examiner public key to verify the Ed25519 signature over content_hash.
`,
		caseData.CaseNumber,
		time.Now().UTC().Format(time.RFC3339),
		examiner,
		appVersion,
		caseData.CaseNumber,
	)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filepath.Base(path), err)
	}
	return os.WriteFile(path, data, 0o600)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func sanitizeTS(ts string) string {
	return reSanitizeTS.ReplaceAllString(ts, "")
}

func sanitizeDirName(s string) string {
	return reSanitizeDirName.ReplaceAllString(s, "_")
}

func defangGo(value, iocType string) string {
	switch iocType {
	case "ipv4":
		return strings.ReplaceAll(value, ".", "[.]")
	case "ipv6":
		return strings.ReplaceAll(value, ":", "[:]")
	case "domain":
		idx := strings.LastIndex(value, ".")
		if idx >= 0 {
			return value[:idx] + "[.]" + value[idx+1:]
		}
		return value
	case "url":
		v := strings.ReplaceAll(value, ".", "[.]")
		v = strings.Replace(v, "https://", "hxxps://", 1)
		v = strings.Replace(v, "http://", "hxxp://", 1)
		v = strings.Replace(v, "ftp://", "fxp://", 1)
		return v
	case "email":
		at := strings.Index(value, "@")
		if at < 0 {
			return value
		}
		domain := strings.ReplaceAll(value[at+1:], ".", "[.]")
		return value[:at] + "[@]" + domain
	default:
		return value
	}
}
