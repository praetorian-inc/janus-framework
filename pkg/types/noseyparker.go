package types

import (
	"fmt"
	"regexp"
	"strings"
)

type CanNPInput interface {
	ToNPInputs() ([]NPInput, error)
}

var RE = regexp.MustCompile(`([^\/]+\/)*(http|https|ssh)((\/[^\/]+)+)`)

func recoverGitURL(input string) string {
	matches := RE.FindStringSubmatch(input)

	if len(matches) >= 4 {
		protocol := matches[2]
		pathPart := matches[3]

		pathElements := strings.Split(strings.TrimPrefix(pathPart, "/"), "/")
		return protocol + "://" + strings.Join(pathElements, "/")
	}
	return input
}

type NPFinding struct {
	FindingID  string       `json:"finding_id"`
	RuleName   string       `json:"rule_name"`
	RuleTextID string       `json:"rule_text_id"`
	Provenance NPProvenance `json:"provenance"`
	Snippet    NPSnippet    `json:"snippet"`
}

type NPOutput struct {
	FindingID  string    `json:"finding_id"`
	RuleName   string    `json:"rule_name"`
	RuleTextID string    `json:"rule_text_id"`
	Matches    []NPMatch `json:"matches"`
}

func (n *NPOutput) ToNPFindings() []NPFinding {
	findings := []NPFinding{}
	for _, match := range n.Matches {
		for i := range match.Provenance {
			provenance, err := match.ProvenanceOf(i)
			if err != nil {
				continue
			}
			if provenance.Kind == "" {
				provenance.Kind = match.Provenance[i].Kind
			}

			if provenance.Kind != "webpage" {
				provenance.RepoPath = recoverGitURL(provenance.RepoPath)
			}

			findings = append(findings, NPFinding{
				FindingID:  n.FindingID,
				RuleName:   n.RuleName,
				RuleTextID: n.RuleTextID,
				Provenance: provenance,
				Snippet:    match.Snippet,
			})
		}
	}
	return findings
}

type NPMatch struct {
	// Noseyparker formats Provenance data differently in the input vs the output
	Provenance []struct {
		Kind         string       `json:"kind,omitempty"`
		NPProvenance              // This is not a typo. Provenance data from NP can exist either directly in each "Provenance" item, or embedded in the "Payload" field.
		Payload      NPProvenance `json:"payload,omitempty"`
	} `json:"provenance"`
	Snippet NPSnippet `json:"snippet"`
}

func (n *NPMatch) ProvenanceOf(index int) (NPProvenance, error) {
	if index >= len(n.Provenance) {
		return NPProvenance{}, fmt.Errorf("index out of bounds")
	}

	provenanceData := n.Provenance[index]
	if provenanceData.Payload != (NPProvenance{}) {
		return provenanceData.Payload, nil
	}

	if provenanceData.NPProvenance != (NPProvenance{}) {
		return provenanceData.NPProvenance, nil
	}

	return NPProvenance{}, fmt.Errorf("no provenance data found")
}

type NPSnippet struct {
	Before   string `json:"before"`
	Matching string `json:"matching"`
	After    string `json:"after"`
}

type NPInput struct {
	ContentBase64 string       `json:"content_base64,omitempty"`
	Content       string       `json:"content,omitempty"`
	Provenance    NPProvenance `json:"provenance"` // see comment above (Noseyparker formats provenance differently in the input vs the output)
}

type NPProvenance struct {
	Kind         string            `json:"kind,omitempty"`
	Platform     string            `json:"platform,omitempty"`
	ResourceType string            `json:"resource_type,omitempty"`
	ResourceID   string            `json:"resource_id,omitempty"`
	Region       string            `json:"region,omitempty"`
	AccountID    string            `json:"account_id,omitempty"`
	FirstCommit  *NPCommitMetadata `json:"first_commit,omitempty"`
	RepoPath     string            `json:"repo_path,omitempty"`
}

type NPCommitMetadata struct {
	CommitMetadata struct {
		CommitID           string `json:"commit_id,omitempty"`
		CommitterName      string `json:"committer_name,omitempty"`
		CommitterEmail     string `json:"committer_email,omitempty"`
		CommitterTimestamp string `json:"committer_timestamp,omitempty"`
		AuthorName         string `json:"author_name,omitempty"`
		AuthorEmail        string `json:"author_email,omitempty"`
		AuthorTimestamp    string `json:"author_timestamp,omitempty"`
		Message            string `json:"message,omitempty"`
	} `json:"commit_metadata,omitempty"`
	BlobPath string `json:"blob_path,omitempty"`
}
