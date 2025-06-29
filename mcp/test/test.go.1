package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer("StreamableHTTP API Server", "1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true),
	)

	// Add RESTful tools
	s.AddTool(
		mcp.NewTool("get_user",
			mcp.WithDescription("Get user information"),
			mcp.WithString("user_id", mcp.Required()),
		),
		handleGetUser,
	)

	s.AddTool(
		mcp.NewTool("create_user",
			mcp.WithDescription("Create a new user"),
			mcp.WithString("name", mcp.Required()),
			mcp.WithString("email", mcp.Required()),
			mcp.WithNumber("age", mcp.Min(0)),
		),
		handleCreateUser,
	)

	s.AddTool(
		mcp.NewTool("search_users",
			mcp.WithDescription("Search users with filters"),
			mcp.WithString("query", mcp.Description("Search query")),
			mcp.WithNumber("limit", mcp.DefaultNumber(10), mcp.Max(100)),
			mcp.WithNumber("offset", mcp.DefaultNumber(0), mcp.Min(0)),
		),
		handleSearchUsers,
	)

	// Add resources
	s.AddResource(
		mcp.NewResource(
			"users://{user_id}",
			"User Profile",
			mcp.WithResourceDescription("User profile data"),
			mcp.WithMIMEType("application/json"),
		),
		handleUserResource,
	)

	// Start StreamableHTTP server
	log.Println("Starting StreamableHTTP server on :8080")
	httpServer := server.NewStreamableHTTPServer(s)
	if err := httpServer.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}

func handleGetUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID := req.Params.Arguments["user_id"].(string)

	// Simulate database lookup
	user, err := getUserFromDB(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return mcp.NewToolResultJSON(user), nil
}

func handleCreateUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.Params.Arguments["name"].(string)
	email := req.Params.Arguments["email"].(string)
	age := int(req.Params.Arguments["age"].(float64))

	// Validate input
	if !isValidEmail(email) {
		return nil, fmt.Errorf("invalid email format: %s", email)
	}

	// Create user
	user := &User{
		ID:        generateID(),
		Name:      name,
		Email:     email,
		Age:       age,
		CreatedAt: time.Now(),
	}

	if err := saveUserToDB(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"id":      user.ID,
		"message": "User created successfully",
		"user":    user,
	}), nil
}

// Helper functions and types for the examples
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
}

func getUserFromDB(userID string) (*User, error) {
	// Placeholder implementation
	return &User{
		ID:    userID,
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}, nil
}

func isValidEmail(email string) bool {
	// Simple email validation
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func generateID() string {
	// Placeholder implementation
	return fmt.Sprintf("user_%d", time.Now().UnixNano())
}

func saveUserToDB(user *User) error {
	// Placeholder implementation
	return nil
}

func handleSearchUsers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := getStringParam(req.Params.Arguments, "query", "")
	limit := int(getFloatParam(req.Params.Arguments, "limit", 10))
	offset := int(getFloatParam(req.Params.Arguments, "offset", 0))

	// Search users with pagination
	users, total, err := searchUsersInDB(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"query":  query,
	}), nil
}

func handleUserResource(ctx context.Context, req mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	userID := extractUserIDFromURI(req.Params.URI)

	user, err := getUserFromDB(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return mcp.NewResourceResultJSON(user), nil
}

// Additional helper functions for parameter handling
func getStringParam(args map[string]interface{}, key, defaultValue string) string {
	if val, ok := args[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getFloatParam(args map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := args[key]; ok && val != nil {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return defaultValue
}

func searchUsersInDB(query string, limit, offset int) ([]*User, int, error) {
	// Placeholder implementation
	users := []*User{
		{ID: "1", Name: "John Doe", Email: "john@example.com", Age: 30},
		{ID: "2", Name: "Jane Smith", Email: "jane@example.com", Age: 25},
	}
	return users, len(users), nil
}

func extractUserIDFromURI(uri string) string {
	// Extract user ID from URI like "users://123"
	parts := strings.Split(uri, "://")
	if len(parts) > 1 {
		return parts[1]
	}
	return uri
}
