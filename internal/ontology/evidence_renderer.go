package ontology

import (
	"fmt"
	"regexp"
	"strings"
)

// EvidenceValues holds the structured numeric and metadata values used for
// rendering evidence interpretation templates.
type EvidenceValues struct {
	ObjectID string
	Current  float64
	Baseline float64
	Delta    float64
	Severity string
}

// RenderedEvidence holds the original source key (e.g. "metric:name" or
// "link:name") together with the rendered interpretation string.
type RenderedEvidence struct {
	Source   string
	Rendered string
}

// RenderEvidence substitutes {placeholder} tokens in template with the
// corresponding values from params. Float64 values are formatted to two
// decimal places ("%.2f"). Any {placeholder} that does not appear in params
// is replaced with "N/A".
func RenderEvidence(template string, params map[string]interface{}) string {
	if template == "" {
		return ""
	}

	if !strings.Contains(template, "{") {
		return template
	}

	// Build replacer pairs for every known key.
	var pairs []string
	for key, val := range params {
		placeholder := "{" + key + "}"
		var strVal string
		switch v := val.(type) {
		case float64:
			strVal = fmt.Sprintf("%.2f", v)
		case string:
			strVal = v
		default:
			strVal = fmt.Sprint(v)
		}
		pairs = append(pairs, placeholder, strVal)
	}

	replacer := strings.NewReplacer(pairs...)
	result := replacer.Replace(template)

	// Replace any remaining {placeholder} patterns with "N/A".
	re := regexp.MustCompile(`\{[^}]*\}`)
	result = re.ReplaceAllString(result, "N/A")

	return result
}

// RenderRecipeEvidence renders the evidence rules from a recipe against the
// corresponding metric values. For each metric evidence rule (source prefixed
// "metric:<name>"), it looks up the metric result by name and substitutes the
// metric's value and baseline into the interpretation template. Link and other
// non-metric rules are rendered with just the object_id.
//
// metricResults is a map of metric name (as used in the recipe Include.Metrics
// list) to the MetricResult. Callers that have the metric results loaded via
// MetricQueryResolver.QueryMetric (or QueryMetrics) should pass them here.
//
// When a metric evidence rule references a metric that is not in metricResults,
// or when metricResults is nil/empty, the rule's current/baseline/delta default
// to zero and {delta} is zero.
//
// Example:
//
//	recipe, _ := registry.GetRecipe("seller_late_delivery_alert")
//	results := map[string]*MetricResult{
//	    "seller_late_delivery_rate_7d": {Value: 0.31, Baseline: 0.08},
//	}
//	rendered := RenderRecipeEvidence(recipe, "SELLER_001", results)
//	// rendered[0].Rendered == "Seller SELLER_001 late delivery rate is 0.31 ..."
func RenderRecipeEvidence(recipe *ContextRecipe, objectID string, severity string, metricResults map[string]*MetricResult) []RenderedEvidence {
	if recipe == nil {
		return nil
	}
	if len(recipe.EvidenceRules) == 0 {
		return []RenderedEvidence{}
	}
	out := make([]RenderedEvidence, 0, len(recipe.EvidenceRules))
	for _, rule := range recipe.EvidenceRules {
		source := rule.Source
		params := map[string]interface{}{
			"object_id": objectID,
			"severity":  severity,
		}

		if strings.HasPrefix(source, "metric:") {
			metricName := strings.TrimPrefix(source, "metric:")
			var current, baseline float64
			if res, ok := metricResults[metricName]; ok && res != nil {
				current = res.Value
				baseline = res.Baseline
			}
			delta := current - baseline
			params["current"] = current
			params["baseline"] = baseline
			params["delta"] = delta
		}

		rendered := RenderEvidence(rule.Interpretation, params)
		out = append(out, RenderedEvidence{
			Source:   source,
			Rendered: rendered,
		})
	}
	return out
}

// RenderAllEvidence renders every EvidenceRule in the slice against the
// supplied structured values and returns the corresponding RenderedEvidence
// entries. Returns nil when rules is nil.
func RenderAllEvidence(rules []EvidenceRule, values EvidenceValues) []RenderedEvidence {
	if rules == nil {
		return nil
	}
	if len(rules) == 0 {
		return []RenderedEvidence{}
	}

	params := map[string]interface{}{
		"object_id": values.ObjectID,
		"current":   values.Current,
		"baseline":  values.Baseline,
		"delta":     values.Delta,
		"severity":  values.Severity,
	}

	out := make([]RenderedEvidence, len(rules))
	for i, rule := range rules {
		out[i] = RenderedEvidence{
			Source:   rule.Source,
			Rendered: RenderEvidence(rule.Interpretation, params),
		}
	}
	return out
}
