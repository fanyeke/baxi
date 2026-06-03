package decision

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"baxi/internal/governance"
	"baxi/internal/llm"
	"baxi/internal/ontology"
	ontRepo "baxi/internal/repository/ontology"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecipeContextBuilder struct {
	caseSvc     DecisionCaseDataProvider
	compiler    *ontology.QueryCompiler
	metricQuery *ontology.MetricQueryResolver
	linkExec    *ontRepo.LinkExecutor
	pool        *pgxpool.Pool
	actionTypes ActionTypeProvider
	recipes     map[string]*ontology.ContextRecipe
}

func NewRecipeContextBuilder(
	caseSvc DecisionCaseDataProvider,
	compiler *ontology.QueryCompiler,
	metricQuery *ontology.MetricQueryResolver,
	linkExec *ontRepo.LinkExecutor,
	pool *pgxpool.Pool,
	actionTypes ActionTypeProvider,
	recipes map[string]*ontology.ContextRecipe,
) *RecipeContextBuilder {
	return &RecipeContextBuilder{
		caseSvc:     caseSvc,
		compiler:    compiler,
		metricQuery: metricQuery,
		linkExec:    linkExec,
		pool:        pool,
		actionTypes: actionTypes,
		recipes:     recipes,
	}
}

func (b *RecipeContextBuilder) BuildEnvelope(ctx context.Context, caseID string, recipeID string) (*llm.LLMSafeContextEnvelope, error) {
	caseRow, err := b.caseSvc.GetCaseByID(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("fetch case %s: %w", caseID, err)
	}

	objectType := derefString(caseRow.ObjectType)
	objectID := derefString(caseRow.ObjectID)
	severity := derefString(caseRow.Severity)
	ruleID := ""
	if caseRow.AlertID != nil && *caseRow.AlertID != "" {
		ruleID = *caseRow.AlertID
	}

	recipe := b.matchRecipe(ruleID, recipeID)
	if recipe == nil {
		return nil, fmt.Errorf("no recipe matched for case %s", caseID)
	}

	rootObj, err := b.loadRootObject(ctx, recipe, objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("load root object: %w", err)
	}

	// Check object type maturity before including in context.
	// stable=full, virtual=with note, planned=skip.
	if rootObj != nil {
		if ot, ok := b.compiler.GetObjectType(objectType); ok {
			switch ot.Maturity {
			case "planned":
				rootObj = nil
			case "virtual":
				if rootObj.Properties == nil {
					rootObj.Properties = make(map[string]interface{})
				}
				rootObj.Properties["_maturity_note"] = "Object type is virtual — derived/computed, not backed by a physical table"
			}
		}
	}

	metricResults := b.loadMetrics(ctx, objectID, recipe)
	linkedObjects := b.loadLinks(ctx, objectType, objectID, recipe)

	props := buildProps(rootObj, metricResults)

	role := recipe.Governance.Role
	govProps := b.applyGovernance(ctx, objectType, props, role, recipe)

	trigger := TriggerInfo{
		Severity: severity,
		AlertID:  safeString(caseRow.AlertID),
	}

	evidence := b.buildEvidence(trigger, metricResults)
	renderedEvidence := ontology.RenderRecipeEvidence(recipe, objectID, severity, metricResults)
	// Convert ontology.RenderedEvidence to llm.RenderedEvidence for the envelope.
	envRenderedEvidence := make([]llm.RenderedEvidence, len(renderedEvidence))
	for i, re := range renderedEvidence {
		envRenderedEvidence[i] = llm.RenderedEvidence{Source: re.Source, Rendered: re.Rendered}
	}
	allowedActions, forbiddenActions := b.resolveActions(recipe)

	classifications := make(map[string]string)
	overallClassification := resolveOverallClassification(classifications)

	govData := llm.GovernanceInfo{
		Classification:   overallClassification,
		RedactionApplied: govProps.RedactionApplied,
		RedactedFields:   govProps.RedactedFields,
		Role:             role,
	}
	dcGovData := GovernanceData{
		Classification:   overallClassification,
		RedactionApplied: govProps.RedactionApplied,
		RedactedFields:   govProps.RedactedFields,
		Role:             role,
	}

	var policyResult *PolicyResult
	if b.actionTypes != nil {
		policyResult = b.buildPolicyResult()
	}
	_ = policyResult

	enrichedObjects := b.buildEnrichedObjects(linkedObjects)

	objectCtx := llm.ObjectContext{
		ObjectType: objectType,
		ObjectID:   objectID,
		Properties: govProps.Properties,
	}
	dcObjectCtx := ObjectContextData{
		ObjectType: objectType,
		ObjectID:   objectID,
		Properties: govProps.Properties,
	}

	llmCtx := &LLMSafeContext{
		CaseID:           caseID,
		Trigger:          trigger,
		ObjectContext:    dcObjectCtx,
		GovernanceInfo:   dcGovData,
		AllowedActions:   allowedActions,
		ForbiddenActions: forbiddenActions,
	}
	contextHash, hashErr := ComputeContextHash(llmCtx)
	if hashErr != nil {
		contextHash = ""
	}

	llmTrigger := llm.TriggerInfo{
		AlertID:       trigger.AlertID,
		RuleID:        trigger.RuleID,
		Severity:      trigger.Severity,
		MetricName:    trigger.MetricName,
		CurrentValue:  trigger.CurrentValue,
		BaselineValue: trigger.BaselineValue,
		DeltaPct:      trigger.DeltaPct,
	}

	envelope := &llm.LLMSafeContextEnvelope{
		SchemaVersion:    "llm_safe_context.v1",
		CaseID:           caseID,
		AlertID:          trigger.AlertID,
		ContextHash:      contextHash,
		BuiltAt:          time.Now(),
		Trigger:          llmTrigger,
		ObjectContext:    objectCtx,
		Evidence:         evidence,
		RenderedEvidence: envRenderedEvidence,
		AllowedActions:   allowedActions,
		ForbiddenActions: forbiddenActions,
		Governance:       govData,
		RedactionSummary: llm.RedactionSummary{
			TotalFields:   len(props),
			RedactedCount: len(govProps.RedactedFields),
			RedactedList:  govProps.RedactedFields,
			AppliedRole:   role,
		},
		PromptVersion: "recipe_v1",
		ConfigVersions: map[string]string{
			"recipe": recipe.Name,
		},
	}

	_ = enrichedObjects

	return envelope, nil
}

func (b *RecipeContextBuilder) matchRecipe(ruleID, recipeID string) *ontology.ContextRecipe {
	if recipeID != "" {
		if r, ok := b.recipes[recipeID]; ok {
			return r
		}
	}
	for _, r := range b.recipes {
		if r.Trigger.RuleID != "" && (r.Trigger.RuleID == ruleID || strings.Contains(ruleID, r.Trigger.RuleID)) {
			return r
		}
	}
	return nil
}

func (b *RecipeContextBuilder) loadRootObject(ctx context.Context, recipe *ontology.ContextRecipe, objectType, objectID string) (*ontRepo.ObjectInstance, error) {
	if objectType == "" || objectID == "" {
		return nil, nil
	}

	cq, err := b.compiler.CompileObjectQuery(objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("compile query for %s: %w", objectType, err)
	}

	rows, err := b.pool.Query(ctx, cq.SQL, cq.Args)
	if err != nil {
		return nil, fmt.Errorf("query object %s/%s: %w", objectType, objectID, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("object %s/%s not found", objectType, objectID)
	}

	values, err := rows.Values()
	if err != nil {
		return nil, fmt.Errorf("scan object %s/%s: %w", objectType, objectID, err)
	}

	props := make(map[string]interface{}, len(cq.Columns))
	for i, col := range cq.Columns {
		if i < len(values) {
			props[col] = values[i]
		}
	}

	id := objectID
	if cq.PrimaryKey != "" {
		if v, ok := props[cq.PrimaryKey]; ok && v != nil {
			id = fmt.Sprintf("%v", v)
		}
	}

	if len(recipe.Include.RootProperties) > 0 {
		filtered := make(map[string]interface{}, len(recipe.Include.RootProperties))
		for _, p := range recipe.Include.RootProperties {
			if v, ok := props[p]; ok {
				filtered[p] = v
			}
		}
		props = filtered
	}

	props = b.filterPropertiesByAvailability(objectType, props)

	return &ontRepo.ObjectInstance{
		ObjectType: objectType,
		ID:         id,
		Properties: props,
	}, nil
}

func (b *RecipeContextBuilder) loadMetrics(ctx context.Context, objectID string, recipe *ontology.ContextRecipe) map[string]*ontology.MetricResult {
	results := make(map[string]*ontology.MetricResult)
	if b.metricQuery == nil {
		return results
	}
	for _, mName := range recipe.Include.Metrics {
		def, ok := b.metricQuery.GetMetric(mName)
		if !ok {
			continue
		}
		res, err := b.metricQuery.QueryMetric(ctx, objectID, def)
		if err != nil {
			log.Printf("warning: query metric %s: %v", mName, err)
			continue
		}
		results[mName] = res
	}
	return results
}

type linkedObject struct {
	LinkName   string
	ObjectType string
	ObjectID   string
	Properties map[string]interface{}
}

func (b *RecipeContextBuilder) loadLinks(ctx context.Context, objectType, objectID string, recipe *ontology.ContextRecipe) []linkedObject {
	if b.linkExec == nil || len(recipe.Include.Links) == 0 {
		return nil
	}

	ot, ok := b.compiler.GetObjectType(objectType)
	if !ok {
		return nil
	}

	var results []linkedObject

	for linkName, linkInclude := range recipe.Include.Links {
		var linkDef *ontology.ObjectLinkV2
		for i := range ot.Links {
			if ot.Links[i].Name == linkName {
				linkDef = &ot.Links[i]
				break
			}
		}
		if linkDef == nil {
			continue
		}

		opts := ontRepo.LinkOptions{
			SourceType:     objectType,
			SourceID:       objectID,
			TargetType:     linkDef.TargetType,
			TargetSchema:   linkDef.Target.Schema,
			TargetTable:    linkDef.Target.Table,
			TargetKey:      linkDef.Target.Key,
			ObjectIDField:  linkDef.Target.ObjectIDField,
			Strategy:       linkDef.Strategy,
			SourceKey:      linkDef.SourceKey,
			Sort:           linkDef.Sort,
		}

		opts.Limit = linkInclude.Limit
		if linkDef.Limit > 0 && opts.Limit <= 0 {
			opts.Limit = linkDef.Limit
		}
		if len(linkInclude.Fields) > 0 {
			opts.Fields = linkInclude.Fields
		} else if len(linkDef.Fields) > 0 {
			opts.Fields = linkDef.Fields
		}

		linked, err := b.linkExec.ExecuteLink(ctx, opts)
		if err != nil {
			log.Printf("warning: execute link %s for %s/%s: %v", linkName, objectType, objectID, err)
			continue
		}

		for i := range linked {
			props := b.filterPropertiesByAvailability(linked[i].ObjectType, linked[i].Properties)
			results = append(results, linkedObject{
				LinkName:   linkName,
				ObjectType: linked[i].ObjectType,
				ObjectID:   linked[i].ID,
				Properties: props,
			})
		}
	}

	return results
}

type govResult struct {
	Properties       map[string]interface{}
	RedactionApplied bool
	RedactedFields   []string
}

func (b *RecipeContextBuilder) applyGovernance(ctx context.Context, objectType string, props map[string]interface{}, role string, recipe *ontology.ContextRecipe) govResult {
	if recipe == nil {
		return govResult{Properties: props}
	}

	redactPII := recipe.Governance.RedactPII
	ot, ok := b.compiler.GetObjectType(objectType)
	classifications := make(map[string]string)
	if ok {
		for name, prop := range ot.Properties {
			switch prop.Sensitivity {
			case "L3":
				classifications[name] = "sensitive"
			case "L2":
				classifications[name] = "internal"
			case "L1":
				classifications[name] = "public_internal"
			default:
				if prop.Sensitivity == "pii" || redactPII {
					classifications[name] = "pii"
				} else {
					classifications[name] = "internal"
				}
			}
		}
	}

	policy := governance.RedactionPolicy{Role: role}
	markings := make(map[string]string)
	redactionResult := governance.RedactObjectContext(props, classifications, markings, policy)

	redactedFieldNames := make([]string, len(redactionResult.RedactedFields))
	for i, rf := range redactionResult.RedactedFields {
		redactedFieldNames[i] = rf.Field
	}

	return govResult{
		Properties:       redactionResult.Properties,
		RedactionApplied: len(redactedFieldNames) > 0,
		RedactedFields:   redactedFieldNames,
	}
}

func (b *RecipeContextBuilder) buildEvidence(trigger TriggerInfo, metricResults map[string]*ontology.MetricResult) []llm.EvidenceItem {
	var items []llm.EvidenceItem
	if trigger.AlertID != "" {
		items = append(items, llm.EvidenceItem{Type: "alert", Key: "alert_id", Value: trigger.AlertID})
	}
	if trigger.RuleID != "" {
		items = append(items, llm.EvidenceItem{Type: "alert", Key: "rule_id", Value: trigger.RuleID})
	}
	if trigger.MetricName != "" {
		items = append(items, llm.EvidenceItem{Type: "metric", Key: "metric_name", Value: trigger.MetricName})
		items = append(items, llm.EvidenceItem{Type: "metric", Key: "current_value", Value: trigger.CurrentValue})
		items = append(items, llm.EvidenceItem{Type: "metric", Key: "baseline_value", Value: trigger.BaselineValue})
		items = append(items, llm.EvidenceItem{Type: "metric", Key: "delta_pct", Value: trigger.DeltaPct})
	}
	for mName, mRes := range metricResults {
		if mRes != nil {
			items = append(items, llm.EvidenceItem{Type: "metric_value", Key: mName+"_value", Value: mRes.Value})
			items = append(items, llm.EvidenceItem{Type: "metric_value", Key: mName+"_baseline", Value: mRes.Baseline})
		}
	}
	if items == nil {
		items = []llm.EvidenceItem{}
	}
	return items
}

func (b *RecipeContextBuilder) resolveActions(recipe *ontology.ContextRecipe) ([]string, []string) {
	allowedActions := make([]string, 0)
	forbiddenActions := make([]string, 0)

	if len(recipe.Include.Actions) > 0 {
		for _, a := range recipe.Include.Actions {
			if b.actionTypes != nil && b.actionTypes.IsActionAllowed(a) {
				allowedActions = append(allowedActions, a)
			} else {
				forbiddenActions = append(forbiddenActions, a)
			}
		}
	} else if b.actionTypes != nil {
		for _, a := range b.actionTypes.ListActionTypes() {
			if b.actionTypes.IsActionAllowed(a) {
				allowedActions = append(allowedActions, a)
			} else {
				forbiddenActions = append(forbiddenActions, a)
			}
		}
	}

	if allowedActions == nil {
		allowedActions = []string{}
	}
	if forbiddenActions == nil {
		forbiddenActions = []string{}
	}
	return allowedActions, forbiddenActions
}

func (b *RecipeContextBuilder) buildEnrichedObjects(linked []linkedObject) []llm.EnrichedObjectData {
	if len(linked) == 0 {
		return nil
	}
	result := make([]llm.EnrichedObjectData, 0, len(linked))
	for _, lo := range linked {
		result = append(result, llm.EnrichedObjectData{
			LinkName:   lo.LinkName,
			Depth:      1,
			ObjectType: lo.ObjectType,
			ObjectID:   lo.ObjectID,
			Properties: lo.Properties,
		})
	}
	return result
}

func (b *RecipeContextBuilder) buildPolicyResult() *PolicyResult {
	if b.actionTypes == nil {
		return nil
	}
	v2 := &ContextBuilderV2{
		caseSvc:     b.caseSvc,
		actionTypes: b.actionTypes,
	}
	return v2.buildPolicyResult()
}

// filterPropertiesByAvailability filters properties based on ObjectPropertyV2.Availability.
// "planned" properties are skipped; "virtual" properties are kept with a source_explanation note.
func (b *RecipeContextBuilder) filterPropertiesByAvailability(objectType string, props map[string]interface{}) map[string]interface{} {
	if b.compiler == nil || len(props) == 0 {
		return props
	}
	ot, ok := b.compiler.GetObjectType(objectType)
	if !ok {
		return props
	}

	filtered := make(map[string]interface{}, len(props))
	var sourceNotes []string

	for name, val := range props {
		propDef, exists := ot.Properties[name]
		if !exists {
			// Property not in schema — keep it as-is
			filtered[name] = val
			continue
		}

		if propDef.Availability == "planned" {
			// Skip planned properties — not yet available
			continue
		}

		filtered[name] = val

		if propDef.Availability == "virtual" {
			sourceNotes = append(sourceNotes, fmt.Sprintf("%s: derived/computed (virtual)", name))
		}
	}

	if len(sourceNotes) > 0 {
		filtered["_source_explanations"] = sourceNotes
	}

	return filtered
}

func buildProps(rootObj *ontRepo.ObjectInstance, metricResults map[string]*ontology.MetricResult) map[string]interface{} {
	props := make(map[string]interface{})
	if rootObj != nil {
		for k, v := range rootObj.Properties {
			props[k] = v
		}
	}
	for mName, mRes := range metricResults {
		if mRes != nil {
			props["metric_"+mName+"_value"] = mRes.Value
			props["metric_"+mName+"_baseline"] = mRes.Baseline
		}
	}
	return props
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
