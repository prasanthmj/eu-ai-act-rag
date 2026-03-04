package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/prasanthmj/eu-ai-act-rag/pipeline"
)

// Handler implements the gomcpgo ToolHandler interface.
type Handler struct {
	pipeline *pipeline.Pipeline
}

// NewHandler creates a new MCP tool handler.
func NewHandler(p *pipeline.Pipeline) *Handler {
	return &Handler{pipeline: p}
}

// ListTools returns the 5 MCP tool definitions.
func (h *Handler) ListTools(ctx context.Context) (*protocol.ListToolsResponse, error) {
	return &protocol.ListToolsResponse{
		Tools: []protocol.Tool{
			{
				Name:        "classify_ai_system",
				Description: "Classifies an AI system's risk tier under the EU AI Act. Returns risk tier, applicable articles, and classification reasoning.",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"description": {
							"type": "string",
							"description": "Plain-English description of the AI system, its purpose, inputs, outputs, and deployment context"
						},
						"domain_hint": {
							"type": "string",
							"enum": ["employment", "education", "biometrics", "critical_infrastructure", "essential_services", "law_enforcement", "migration", "justice", "general_purpose", "unknown"],
							"description": "Optional: known domain"
						}
					},
					"required": ["description"]
				}`),
			},
			{
				Name:        "get_obligations",
				Description: "Returns all compliance obligations under the EU AI Act for a given risk tier, with article citations.",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"risk_tier": {
							"type": "string",
							"enum": ["HIGH_RISK", "LIMITED_RISK", "MINIMAL_RISK", "GPAI"]
						},
						"domain": {
							"type": "string",
							"description": "Optional: specific Annex III domain to filter obligations"
						}
					},
					"required": ["risk_tier"]
				}`),
			},
			{
				Name:        "check_prohibited",
				Description: "Checks if an AI system use-case matches prohibited AI practices under Article 5 of the EU AI Act.",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"description": {
							"type": "string",
							"description": "Description of the AI system's purpose and use-case"
						}
					},
					"required": ["description"]
				}`),
			},
			{
				Name:        "lookup_article",
				Description: "Retrieves the full text and cross-references for a specific EU AI Act article, recital, or annex.",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"reference": {
							"type": "string",
							"description": "Reference like 'article_6', 'recital_48', or 'annex_3'"
						}
					},
					"required": ["reference"]
				}`),
			},
			{
				Name:        "get_compliance_checklist",
				Description: "Runs the full EU AI Act compliance analysis pipeline. Returns risk classification, obligations, confidence score, and a compliance checklist.",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"description": {
							"type": "string",
							"description": "Plain-English description of the AI system"
						},
						"output_format": {
							"type": "string",
							"enum": ["markdown", "json"],
							"default": "markdown"
						}
					},
					"required": ["description"]
				}`),
			},
		},
	}, nil
}

// CallTool dispatches tool calls to the pipeline.
func (h *Handler) CallTool(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResponse, error) {
	switch req.Name {
	case "classify_ai_system":
		return h.classifyAISystem(ctx, req.Arguments)
	case "get_obligations":
		return h.getObligations(ctx, req.Arguments)
	case "check_prohibited":
		return h.checkProhibited(ctx, req.Arguments)
	case "lookup_article":
		return h.lookupArticle(ctx, req.Arguments)
	case "get_compliance_checklist":
		return h.getComplianceChecklist(ctx, req.Arguments)
	default:
		return errorResponse(fmt.Sprintf("unknown tool: %s", req.Name)), nil
	}
}

func (h *Handler) classifyAISystem(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	description, _ := args["description"].(string)
	domainHint, _ := args["domain_hint"].(string)

	result, err := h.pipeline.ClassifySystem(ctx, description, domainHint)
	if err != nil {
		return errorResponse(err.Error()), nil
	}
	return jsonResponse(result)
}

func (h *Handler) getObligations(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	riskTier, _ := args["risk_tier"].(string)
	domain, _ := args["domain"].(string)

	result, err := h.pipeline.GetObligations(ctx, riskTier, domain)
	if err != nil {
		return errorResponse(err.Error()), nil
	}
	return jsonResponse(result)
}

func (h *Handler) checkProhibited(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	description, _ := args["description"].(string)

	result, err := h.pipeline.CheckProhibited(ctx, description)
	if err != nil {
		return errorResponse(err.Error()), nil
	}
	return textResponse(result), nil
}

func (h *Handler) lookupArticle(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	reference, _ := args["reference"].(string)

	result, err := h.pipeline.LookupArticle(ctx, reference)
	if err != nil {
		return errorResponse(err.Error()), nil
	}
	return jsonResponse(result)
}

func (h *Handler) getComplianceChecklist(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	description, _ := args["description"].(string)
	outputFormat, _ := args["output_format"].(string)

	result, err := h.pipeline.RunFull(ctx, description, "")
	if err != nil {
		return errorResponse(err.Error()), nil
	}

	if outputFormat == "json" {
		return jsonResponse(result)
	}
	return textResponse(result.Checklist), nil
}

func textResponse(text string) *protocol.CallToolResponse {
	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{{Type: "text", Text: text}},
	}
}

func jsonResponse(v any) (*protocol.CallToolResponse, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errorResponse(fmt.Sprintf("marshal response: %v", err)), nil
	}
	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{{Type: "text", Text: string(data)}},
	}, nil
}

func errorResponse(msg string) *protocol.CallToolResponse {
	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{{Type: "text", Text: "Error: " + msg}},
		IsError: true,
	}
}
