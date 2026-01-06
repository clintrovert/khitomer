package types

// RepositoryInfo contains GitHub repository information
type RepositoryInfo struct {
	Owner         string
	Name          string
	BaseBranch    string
	FeatureBranch string
	CloneURL      string
}

// PRInfo contains pull request information
type PRInfo struct {
	PRNumber    int64
	PRURL       string
	Title       string
	Description string
	Status      string
}
