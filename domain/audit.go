package domain

// ConsistencyAuditor defines the interface for auditing consistency across generated specifications.
type ConsistencyAuditor interface {
	Audit(files map[string]string) (*ConsistencyReport, error)
}
