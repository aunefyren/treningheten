package controllers

import (
	"context"
	"net/http"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpPrincipalKey struct{}

const mcpDefaultLimit = 20

// --- Tool argument and output types (schemas are inferred from these) ---

type mcpEmptyArgs struct{}

type mcpListWeightsArgs struct {
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of entries to return (default 20)"`
}

type mcpListExercisesArgs struct {
	Action string `json:"action,omitempty" jsonschema:"filter to one exercise type, e.g. Run or Bicycling"`
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum number of activities to return (default 20)"`
}

type mcpWeightsOutput struct {
	Weights []models.MCPWeight `json:"weights"`
}

type mcpLatestWeightOutput struct {
	Weight *models.MCPWeight `json:"weight" jsonschema:"the most recent weight entry, or null if none"`
}

type mcpActivitiesOutput struct {
	Activities []models.MCPActivity `json:"activities"`
}

// MCPHandler authenticates the caller, then serves the MCP Streamable HTTP
// endpoint. The authenticated principal is passed to the per-request server via
// the request context.
func MCPHandler() gin.HandlerFunc {
	streamHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		principal, _ := r.Context().Value(mcpPrincipalKey{}).(middlewares.Principal)
		return buildMCPServer(principal.UserID)
	}, nil)

	return func(c *gin.Context) {
		principal, err := middlewares.Authenticate(c.GetHeader("Authorization"))
		if err != nil {
			// 401 with a WWW-Authenticate challenge so MCP clients discover the OAuth flow.
			middlewares.BearerChallenge(c, http.StatusUnauthorized, "invalid_token", "authentication required")
			return
		}
		if !auth.ScopeCanRead(principal.Scope) {
			middlewares.BearerChallenge(c, http.StatusForbidden, "insufficient_scope", "token lacks API read access")
			return
		}

		ctx := context.WithValue(c.Request.Context(), mcpPrincipalKey{}, principal)
		c.Request = c.Request.WithContext(ctx)
		streamHandler.ServeHTTP(c.Writer, c.Request)
	}
}

// buildMCPServer constructs an MCP server whose tools operate on a single user.
func buildMCPServer(userID uuid.UUID) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "treningheten",
		Version: files.ConfigFile.TreninghetenVersion,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "whoami",
		Description: "Get the authenticated user's profile (name, email, admin status, member since).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpEmptyArgs) (*mcp.CallToolResult, models.MCPProfile, error) {
		profile, err := assembleUserProfile(userID)
		if err != nil {
			return nil, models.MCPProfile{}, err
		}
		return nil, profile, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_weights",
		Description: "List the user's recorded body-weight entries, newest first.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpListWeightsArgs) (*mcp.CallToolResult, mcpWeightsOutput, error) {
		weights, err := assembleUserWeights(userID, limitOrDefault(args.Limit))
		if err != nil {
			return nil, mcpWeightsOutput{}, err
		}
		return nil, mcpWeightsOutput{Weights: weights}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_latest_weight",
		Description: "Get the user's most recent body-weight entry.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpEmptyArgs) (*mcp.CallToolResult, mcpLatestWeightOutput, error) {
		weights, err := assembleUserWeights(userID, 1)
		if err != nil {
			return nil, mcpLatestWeightOutput{}, err
		}
		out := mcpLatestWeightOutput{}
		if len(weights) > 0 {
			out.Weight = &weights[0]
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_exercises",
		Description: "List the user's logged exercise activities (with action type, duration and per-set distance/time/reps/weight), newest first. Optionally filter by exercise type (e.g. Run).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpListExercisesArgs) (*mcp.CallToolResult, mcpActivitiesOutput, error) {
		activities, err := assembleUserActivities(userID, args.Action, limitOrDefault(args.Limit))
		if err != nil {
			return nil, mcpActivitiesOutput{}, err
		}
		return nil, mcpActivitiesOutput{Activities: activities}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_statistics",
		Description: "Get a summary of the user's training: activity counts for the past month, past year and all time, plus total logged distance and time.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpEmptyArgs) (*mcp.CallToolResult, models.MCPStatistics, error) {
		stats, err := assembleUserStatistics(userID)
		if err != nil {
			return nil, models.MCPStatistics{}, err
		}
		return nil, stats, nil
	})

	return server
}

func limitOrDefault(limit int) int {
	if limit <= 0 {
		return mcpDefaultLimit
	}
	return limit
}
