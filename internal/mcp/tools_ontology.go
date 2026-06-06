package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerOntologyTools registers all ontology-related MCP tools.
func (s *Server) registerOntologyTools() {
	// Tool: describe_schema (was describe_ontology)
	describeSchemaTool := mcp.NewTool(
		ToolDescribeSchema,
		mcp.WithDescription("Describe all registered data object types with their properties, relationships, and allowed actions"),
	)
	s.server.AddTool(describeSchemaTool, s.handleDescribeOntology)

	if isLegacyToolsEnabled() {
		legacyDescribeOntologyTool := mcp.NewTool(
			LegacyDescribeOntology,
			mcp.WithDescription("Describe all registered AIP object types with their properties, links, and allowed actions"),
		)
		s.server.AddTool(legacyDescribeOntologyTool, s.handleDescribeOntology)
	}

	// Tool: get_record (was get_object)
	getRecordTool := mcp.NewTool(
		ToolGetRecord,
		mcp.WithDescription("Get a single record by type and ID"),
		mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of record to retrieve")),
		mcp.WithString("object_id", mcp.Required(), mcp.Description("The ID of the record to retrieve")),
	)
	s.server.AddTool(getRecordTool, s.handleGetObject)

	if isLegacyToolsEnabled() {
		legacyGetObjectTool := mcp.NewTool(
			LegacyGetObject,
			mcp.WithDescription("Get a single object by type and ID"),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of object to retrieve")),
			mcp.WithString("object_id", mcp.Required(), mcp.Description("The ID of the object to retrieve")),
		)
		s.server.AddTool(legacyGetObjectTool, s.handleGetObject)
	}

	// Tool: get_related_records (was get_linked_objects)
	getRelatedRecordsTool := mcp.NewTool(
		ToolGetRelatedRecords,
		mcp.WithDescription("Get records linked to a given record via relationships"),
		mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of the source record")),
		mcp.WithString("object_id", mcp.Required(), mcp.Description("The ID of the source record")),
		mcp.WithString("link_name", mcp.Description("Optional: filter by link name")),
		mcp.WithNumber("max_depth", mcp.Description("Optional: traversal depth (default: 1, max: 3)")),
	)
	s.server.AddTool(getRelatedRecordsTool, s.handleGetLinkedObjects)

	if isLegacyToolsEnabled() {
		legacyGetLinkedObjectsTool := mcp.NewTool(
			LegacyGetLinkedObjects,
			mcp.WithDescription("Get objects linked to a given object via relationships"),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of the source object")),
			mcp.WithString("object_id", mcp.Required(), mcp.Description("The ID of the source object")),
			mcp.WithString("link_name", mcp.Description("Optional: filter by link name")),
			mcp.WithNumber("max_depth", mcp.Description("Optional: traversal depth (default: 1, max: 3)")),
		)
		s.server.AddTool(legacyGetLinkedObjectsTool, s.handleGetLinkedObjects)
	}

	// Tool: apply_action (was execute_action)
	applyActionTool := mcp.NewTool(
		ToolApplyAction,
		mcp.WithDescription("Execute an action on a record"),
		mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of the target record")),
		mcp.WithString("object_id", mcp.Required(), mcp.Description("The ID of the target record")),
		mcp.WithString("action_type", mcp.Required(), mcp.Description("The action type to execute")),
		mcp.WithString("params", mcp.Description("Optional JSON-encoded parameters for the action")),
		mcp.WithBoolean("dry_run", mcp.Description("When true (default), simulate execution without side effects")),
	)
	s.server.AddTool(applyActionTool, s.handleExecuteAction)

	if isLegacyToolsEnabled() {
		legacyExecuteActionTool := mcp.NewTool(
			LegacyExecuteAction,
			mcp.WithDescription("Execute an action on an object"),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of the target object")),
			mcp.WithString("object_id", mcp.Required(), mcp.Description("The ID of the target object")),
			mcp.WithString("action_type", mcp.Required(), mcp.Description("The action type to execute")),
			mcp.WithString("params", mcp.Description("Optional JSON-encoded parameters for the action")),
			mcp.WithBoolean("dry_run", mcp.Description("When true (default), simulate execution without side effects")),
		)
		s.server.AddTool(legacyExecuteActionTool, s.handleExecuteAction)
	}
}

// handleDescribeOntology returns metadata for all registered object types.
func (s *Server) handleDescribeOntology(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	descriptor, err := s.ontologySvc.DescribeOntology(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to describe ontology: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"object_types": descriptor.ObjectTypes,
	})
}

// handleGetObject retrieves a single object by type and ID, including metrics.
func (s *Server) handleGetObject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	objectType, ok := args["object_type"].(string)
	if !ok || objectType == "" {
		return mcp.NewToolResultError("object_type is required"), nil
	}

	objectID, ok := args["object_id"].(string)
	if !ok || objectID == "" {
		return mcp.NewToolResultError("object_id is required"), nil
	}

	obj, err := s.ontologySvc.GetObject(ctx, objectType, objectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get object: %v", err)), nil
	}

	// Attempt to fetch metrics; non-fatal if unavailable.
	metrics, _ := s.ontologySvc.GetObjectMetrics(ctx, objectType, objectID)

	result := map[string]interface{}{
		"object_type": obj.ObjectType,
		"object_id":   obj.ObjectID,
		"properties":  obj.Properties,
	}
	if metrics != nil && len(metrics) > 0 {
		result["metrics"] = metrics
	}

	return mcp.NewToolResultJSON(result)
}

// handleGetLinkedObjects retrieves objects linked to the specified object.
func (s *Server) handleGetLinkedObjects(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	objectType, ok := args["object_type"].(string)
	if !ok || objectType == "" {
		return mcp.NewToolResultError("object_type is required"), nil
	}

	objectID, ok := args["object_id"].(string)
	if !ok || objectID == "" {
		return mcp.NewToolResultError("object_id is required"), nil
	}

	linkName, _ := args["link_name"].(string)

	maxDepth := 1
	if depthRaw, ok := args["max_depth"].(float64); ok {
		maxDepth = int(depthRaw)
		if maxDepth < 1 {
			maxDepth = 1
		}
		if maxDepth > 3 {
			return mcp.NewToolResultError("max_depth must not exceed 3"), nil
		}
	}

	result, err := s.ontologySvc.GetLinkedObjects(ctx, objectType, objectID, linkName, maxDepth)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get linked objects: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"object_type": result.ObjectType,
		"object_id":   result.ObjectID,
		"links":       result.Links,
	})
}

// handleExecuteAction executes an action on an object.
func (s *Server) handleExecuteAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	objectType, ok := args["object_type"].(string)
	if !ok || objectType == "" {
		return mcp.NewToolResultError("object_type is required"), nil
	}

	objectID, ok := args["object_id"].(string)
	if !ok || objectID == "" {
		return mcp.NewToolResultError("object_id is required"), nil
	}

	actionType, ok := args["action_type"].(string)
	if !ok || actionType == "" {
		return mcp.NewToolResultError("action_type is required"), nil
	}

	var params map[string]interface{}
	if paramsRaw, ok := args["params"].(string); ok && paramsRaw != "" {
		if err := json.Unmarshal([]byte(paramsRaw), &params); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid JSON in params: %v", err)), nil
		}
	}

	dryRun := true
	if v, ok := args["dry_run"].(bool); ok {
		dryRun = v
	}

	if !dryRun {
		return mcp.NewToolResultError("execute_action with dry_run=false requires an approved proposal. Use propose_action first, then execute_proposal after approval"), nil
	}

	result, err := s.ontologySvc.ExecuteAction(ctx, objectType, objectID, actionType, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to execute action: %v", err)), nil
	}

	res := map[string]interface{}{
		"success":     result.Success,
		"action_type": result.ActionType,
		"object_type": result.ObjectType,
		"object_id":   result.ObjectID,
		"result":      result.Result,
		"dry_run":     dryRun,
	}
	return mcp.NewToolResultJSON(res)
}
