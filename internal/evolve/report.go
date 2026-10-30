package evolve

// Publication and observation states. There is deliberately no ready state: a
// declaration is not evidence, so prepare and plan report not_ready.
const (
	publicationNotReady  = "not_ready"
	publicationSubmitted = "submitted"
	checkNotRun          = "not_run"
	declaredUnverified   = "declared_not_verified"
)

// FileSummary describes one replacement without carrying its content.
type FileSummary struct {
	Path           string `json:"path"`
	PreimageSHA256 string `json:"preimageSha256,omitempty"`
	SHA256         string `json:"sha256"`
	Bytes          int    `json:"bytes"`
}

// Check is one named required check and what zt observed about it. Every status
// this slice produces is not_run: the name is descriptive and no command is
// planned, executed or timed.
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PrepareResult is what prepare wrote and what it validated on the way.
type PrepareResult struct {
	SchemaVersion   int           `json:"schemaVersion"`
	CandidateID     string        `json:"candidateId"`
	CandidatePath   string        `json:"candidatePath"`
	Created         bool          `json:"created"`
	Skill           string        `json:"skill"`
	Target          Target        `json:"target"`
	PolicyDigest    string        `json:"policyDigest"`
	Files           []FileSummary `json:"files"`
	Checks          []Check       `json:"checks"`
	Provenance      Provenance    `json:"provenance"`
	ProvenanceState string        `json:"provenanceState"`
	Evidence        []EvidenceRef `json:"evidence,omitempty"`
	EvidenceState   string        `json:"evidenceState,omitempty"`
	Publication     string        `json:"publication"`
}

// PlanResult is what plan re-read and re-validated. It reports every declaration
// as a declaration.
type PlanResult struct {
	SchemaVersion   int           `json:"schemaVersion"`
	CandidateID     string        `json:"candidateId"`
	CandidatePath   string        `json:"candidatePath"`
	Skill           string        `json:"skill"`
	Target          Target        `json:"target"`
	PolicyDigest    string        `json:"policyDigest"`
	Files           []FileSummary `json:"files"`
	Checks          []Check       `json:"checks"`
	Provenance      Provenance    `json:"provenance"`
	ProvenanceState string        `json:"provenanceState"`
	Evidence        []EvidenceRef `json:"evidence,omitempty"`
	EvidenceState   string        `json:"evidenceState,omitempty"`
	Publication     string        `json:"publication"`
}

func summarizeFiles(files []File) []FileSummary {
	result := make([]FileSummary, len(files))
	for i, file := range files {
		result[i] = FileSummary{
			Path:           file.Path,
			PreimageSHA256: file.PreimageSHA256,
			SHA256:         file.SHA256,
			Bytes:          file.Bytes,
		}
	}
	return result
}

// describeChecks reports the policy's named checks as not_run.
func describeChecks(checks []string) []Check {
	result := make([]Check, len(checks))
	for i, name := range checks {
		result[i] = Check{Name: name, Status: checkNotRun}
	}
	return result
}
