package ontology

// ──── Metric Definition types ─────────────────────────────────────────────────
// These types are loaded from config/metric_definitions.yml and define the
// contract for pre-aggregated metrics used in ContextRecipe and MetricResolver.

// MetricDefinition defines a pre-aggregated metric contract.
// Each metric is bound to an object type and grain, with a physical source
// table, filter criteria, value/baseline columns, optional severity rules,
// and an LLM-readable explanation.
type MetricDefinition struct {
	Name           string
	DisplayName    string
	ObjectType     string
	Grain          string
	Source         MetricSource
	Filters        map[string]string
	ValueColumn    string
	BaselineColumn string
	Severity       map[string]string // e.g. {"medium": "current_value > baseline + 0.10"}
	LLMExplanation string
}

// MetricSource defines the physical table for a metric definition.
type MetricSource struct {
	Schema string
	Table  string
}

// ──── YAML parsing types ─────────────────────────────────────────────────────

type metricDefsConfig struct {
	Version string                  `yaml:"version"`
	Metrics map[string]*rawMetricV1 `yaml:"metrics"`
}

type rawMetricV1 struct {
	DisplayName     string            `yaml:"display_name"`
	ObjectType      string            `yaml:"object_type"`
	Grain           string            `yaml:"grain"`
	Source          rawMetricSourceV1 `yaml:"source"`
	Filters         map[string]string `yaml:"filters,omitempty"`
	ValueColumn     string            `yaml:"value_column"`
	BaselineColumn  string            `yaml:"baseline_column"`
	Severity        map[string]string `yaml:"severity,omitempty"`
	LLMExplanation  string            `yaml:"llm_explanation,omitempty"`
}

type rawMetricSourceV1 struct {
	Schema string `yaml:"schema"`
	Table  string `yaml:"table"`
}

// MetricResolver resolves metric definitions from the metrics config file.
type MetricResolver struct {
	metrics map[string]*MetricDefinition
}

// NewMetricResolver creates a MetricResolver from parsed metric definitions.
func NewMetricResolver(metrics map[string]*MetricDefinition) *MetricResolver {
	return &MetricResolver{metrics: metrics}
}

// GetMetric returns the metric definition for the given name.
func (r *MetricResolver) GetMetric(name string) (*MetricDefinition, bool) {
	m, ok := r.metrics[name]
	return m, ok
}

// ListMetrics returns all metric names.
func (r *MetricResolver) ListMetrics() []string {
	names := make([]string, 0, len(r.metrics))
	for n := range r.metrics {
		names = append(names, n)
	}
	return names
}
