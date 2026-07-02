package context_sanitizer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"STfreApi/common"
	"STfreApi/model"
)

type Finding struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Score    int    `json:"score"`
	Path     string `json:"path,omitempty"`
	Evidence string `json:"evidence,omitempty"`
	Action   string `json:"action,omitempty"`
}

type EventInput struct {
	RequestId      string
	UserId         int
	TokenId        int
	TokenName      string
	ChannelId      int
	ChannelName    string
	RequestedModel string
	MappedModel    string
	Policy         ResolvedPolicy
	Direction      string
	Stage          string
	Action         string
	RiskScore      int
	Findings       []Finding
	SnippetSource  string
	LatencyMs      int64
	Degraded       bool
	CircuitState   string
	Error          string
}

func RecordEventAsync(input EventInput) {
	if input.Policy.Policy == nil || !input.Policy.Config.Logging.LogEvents {
		return
	}
	go func() { _ = RecordEvent(input) }()
}

func RecordEvent(input EventInput) error {
	categories := uniqueFindingTypes(input.Findings)
	categoriesJSON, _ := json.Marshal(categories)
	findingsJSON, _ := json.Marshal(input.Findings)
	maxSnippet := input.Policy.Config.Logging.LogSnippetChars
	if maxSnippet <= 0 {
		maxSnippet = 160
	}
	snippet := ""
	if input.Policy.Config.Logging.LogRawContent {
		snippet = BuildSnippet(input.SnippetSource, maxSnippet)
	}
	contentHash := ""
	if input.Policy.Config.Logging.HashContent && input.SnippetSource != "" {
		contentHash = HashContent(input.SnippetSource)
	}
	policyId := 0
	policyVersion := 0
	mode := ModeOff
	if input.Policy.Policy != nil {
		policyId = input.Policy.Policy.Id
		policyVersion = input.Policy.Policy.Version
		mode = input.Policy.Policy.Mode
	}
	return common.DB.Create(&model.ContextSanitizationEvent{
		CreatedAt:      time.Now().Unix(),
		RequestId:      input.RequestId,
		UserId:         input.UserId,
		TokenId:        input.TokenId,
		TokenName:      input.TokenName,
		ChannelId:      input.ChannelId,
		ChannelName:    input.ChannelName,
		RequestedModel: input.RequestedModel,
		MappedModel:    input.MappedModel,
		PolicyId:       policyId,
		PolicyVersion:  policyVersion,
		Direction:      input.Direction,
		Stage:          input.Stage,
		Mode:           mode,
		Action:         input.Action,
		RiskScore:      input.RiskScore,
		Categories:     string(categoriesJSON),
		Findings:       string(findingsJSON),
		Snippet:        snippet,
		ContentHash:    contentHash,
		LatencyMs:      input.LatencyMs,
		Degraded:       input.Degraded,
		CircuitState:   input.CircuitState,
		Error:          BuildSnippet(input.Error, 500),
	}).Error
}

func HashContent(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func BuildSnippet(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 || utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes])
}

func uniqueFindingTypes(findings []Finding) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, f := range findings {
		if f.Type == "" || seen[f.Type] {
			continue
		}
		seen[f.Type] = true
		out = append(out, f.Type)
	}
	return out
}
