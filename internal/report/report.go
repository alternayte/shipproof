package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/shipproof/shipproof/internal/review"
	"github.com/shipproof/shipproof/internal/schema"
)

func GenerateChangeReport(w io.Writer, root, changeID string) error {
	pack, err := loadEvidencePack(root, changeID)
	if err != nil {
		return err
	}

	var packet *review.ReviewPacket
	if rp, err := review.Prepare(root, changeID); err == nil {
		packet = &rp
	}

	generatedAt := pack.Provenance.GeneratedAt

	data := changeReportData{
		ChangeID:    pack.ChangeID,
		GeneratedAt: generatedAt,
		Intent:      buildIntentData(pack),
		Verify:      buildVerifyData(pack, packet),
		Implement:   buildImplementData(pack),
		AgentRun:    buildAgentRunData(pack),
		Provenance:  buildReportProvenanceData(pack),
	}

	return executeTemplate(w, "change_report.html", data)
}

func GeneratePRSummary(w io.Writer, root, changeID string) error {
	pack, err := loadEvidencePack(root, changeID)
	if err != nil {
		return err
	}

	packet, err := review.Prepare(root, changeID)
	if err != nil {
		return fmt.Errorf("prepare review packet: %w", err)
	}

	return writePRSummary(w, pack, packet)
}

func loadEvidencePack(root, changeID string) (schema.EvidencePack, error) {
	packPath := filepath.Join(root, ".shipproof", "changes", changeID, "evidence-pack.json")

	data, err := os.ReadFile(packPath)
	if err != nil {
		if os.IsNotExist(err) {
			return schema.EvidencePack{}, fmt.Errorf("evidence pack not found for change %q; run shipproof evidence pack %s first", changeID, changeID)
		}
		return schema.EvidencePack{}, fmt.Errorf("read evidence pack: %w", err)
	}

	var pack schema.EvidencePack
	if err := json.Unmarshal(data, &pack); err != nil {
		return schema.EvidencePack{}, fmt.Errorf("parse evidence pack: %w", err)
	}

	return pack, nil
}
