package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"github.com/digitalghost404/inkandbone/internal/db"
)

var errInvalidPDF = errors.New("invalid or unreadable PDF")

func (s *Server) handleListRulebookSources(w http.ResponseWriter, r *http.Request) {
	rulesetID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid ruleset id", http.StatusBadRequest)
		return
	}
	sources, err := s.db.ListRulebookSources(rulesetID)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if sources == nil {
		sources = []db.RulebookSource{}
	}
	writeJSON(w, sources)
}

func (s *Server) handleIngestRulebook(w http.ResponseWriter, r *http.Request) {
	rulesetID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid ruleset id", http.StatusBadRequest)
		return
	}

	ct := r.Header.Get("Content-Type")

	var text string
	var source string

	switch {
	case strings.HasPrefix(ct, "text/plain"):
		source = r.URL.Query().Get("source")
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		text = string(b)

	case strings.HasPrefix(ct, "multipart/form-data"):
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
			return
		}
		source = r.FormValue("source")
		file, _, err := r.FormFile("rulebook")
		if err != nil {
			http.Error(w, "rulebook field is required: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		extracted, err := extractPDFText(file)
		if err != nil {
			if errors.Is(err, errInvalidPDF) {
				http.Error(w, "pdf extraction: "+err.Error(), http.StatusUnprocessableEntity)
			} else {
				http.Error(w, "pdf extraction: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		text = extracted

	default:
		http.Error(w, "unsupported Content-Type; use text/plain or multipart/form-data", http.StatusUnsupportedMediaType)
		return
	}

	if source == "" {
		source = "Core Rulebook"
	}

	ruleset, err := s.db.GetRuleset(rulesetID)
	if err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var rulesetName string
	if ruleset != nil {
		rulesetName = ruleset.Name
	}

	chunks := chunkRulebook(rulesetName, text, source)
	// Replace only chunks for this source — other books are untouched.
	if err := s.db.DeleteRulebookChunksBySource(rulesetID, source); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.db.CreateRulebookChunks(rulesetID, chunks); err != nil {
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{"chunks_created": len(chunks), "source": source})
}

// chunkRulebook dispatches to the appropriate chunking strategy for the given ruleset.
// Each TTRPG system may have different source PDF structure, so chunking is per-ruleset.
func chunkRulebook(rulesetName, text, source string) []db.RulebookChunk {
	switch strings.ToLower(rulesetName) {
	case "vtm", "dune":
		// These systems' PDFs extract as plain prose — no markdown headings in the extracted text.
		// Split on paragraph boundaries up to a max chunk size.
		return chunkByParagraphs(text, source)
	default:
		// All other rulesets: split on markdown "#" heading lines (user-supplied or structured text).
		return chunkByHeadingsWithSource(text, source)
	}
}

// chunkByHeadingsWithSource splits text into chunks delimited by lines starting with "#",
// tagging each chunk with the given source book name.
// If no heading lines are found, the entire text becomes one chunk with an empty heading.
func chunkByHeadingsWithSource(text, source string) []db.RulebookChunk {
	var chunks []db.RulebookChunk
	var currentHeading strings.Builder
	var currentContent strings.Builder

	flush := func() {
		if currentHeading.Len() > 0 || currentContent.Len() > 0 {
			chunks = append(chunks, db.RulebookChunk{
				Source:  source,
				Heading: strings.TrimSpace(strings.TrimLeft(currentHeading.String(), "#")),
				Content: strings.TrimSpace(currentContent.String()),
			})
			currentHeading.Reset()
			currentContent.Reset()
		}
	}

	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "#") {
			flush()
			currentHeading.WriteString(line)
		} else {
			currentContent.WriteString(line + "\n")
		}
	}
	flush()
	return chunks
}

// chunkByParagraphs splits plain text (no markdown headings) on blank lines,
// grouping paragraphs into chunks up to maxChunkRunes characters, with
// overlapRunes of trailing content carried into the next chunk so that rules
// spanning a chunk boundary are not silently truncated.
func chunkByParagraphs(text, source string) []db.RulebookChunk {
	const maxChunkRunes = 3000
	const overlapRunes = 500

	var chunks []db.RulebookChunk

	// Collect non-empty paragraphs.
	var paras []string
	for _, p := range strings.Split(text, "\n\n") {
		p = strings.TrimSpace(p)
		if p != "" {
			paras = append(paras, p)
		}
	}

	var current strings.Builder
	var overlap string // tail of the previous chunk to prepend

	flushChunk := func() {
		s := strings.TrimSpace(current.String())
		if s == "" {
			return
		}
		chunks = append(chunks, db.RulebookChunk{
			Source:  source,
			Heading: "",
			Content: s,
		})
		// Carry the last overlapRunes runes into the next chunk.
		runes := []rune(s)
		if len(runes) > overlapRunes {
			overlap = string(runes[len(runes)-overlapRunes:])
		} else {
			overlap = s
		}
		current.Reset()
	}

	for _, para := range paras {
		if current.Len() > 0 && current.Len()+len(para) > maxChunkRunes {
			flushChunk()
			// Seed next chunk with overlap from previous chunk.
			if overlap != "" {
				current.WriteString(overlap)
				overlap = ""
			}
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}
	flushChunk()

	return chunks
}

// extractPDFText validates the PDF then extracts text using pdftotext (poppler),
// which correctly handles CIDFont/hex-encoded PDFs used by commercial TTRPG books.
func extractPDFText(r io.ReadSeeker) (string, error) {
	conf := model.NewDefaultConfiguration()
	if err := pdfapi.Validate(r, conf); err != nil {
		return "", fmt.Errorf("%w: %v", errInvalidPDF, err)
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	// Write to a temp file so pdftotext can read it by path.
	tmp, err := os.CreateTemp("", "rulebook-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()

	// pdftotext with "-" as output writes to stdout.
	out, err := exec.Command("pdftotext", tmp.Name(), "-").Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w (install poppler-utils)", err)
	}
	return string(out), nil
}
