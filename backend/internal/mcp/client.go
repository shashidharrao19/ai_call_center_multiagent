package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client represents an MCP client for function calling and data access
type Client struct {
	serverURL          string
	httpClient         *resty.Client
	isConnected        bool
	availableFunctions []map[string]interface{}
	mu                 sync.RWMutex
}

// NewClient creates a new MCP client
func NewClient(serverURL string) *Client {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)

	return &Client{
		serverURL:  serverURL,
		httpClient: client,
	}
}

// Connect connects to the MCP server
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get available functions from MCP server
	if err := c.getAvailableFunctions(); err != nil {
		log.Printf("Warning: Failed to get available functions: %v", err)
		// Continue without functions - MCP might not be available
	}

	c.isConnected = true
	log.Printf("Connected to MCP server at %s", c.serverURL)
	return nil
}

// getAvailableFunctions gets list of available functions from MCP server
func (c *Client) getAvailableFunctions() error {
	resp, err := c.httpClient.R().
		Get(c.serverURL + "/functions")

	if err != nil {
		return fmt.Errorf("failed to get available functions: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		log.Printf("MCP server returned status %d", resp.StatusCode())
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &data); err != nil {
		return fmt.Errorf("failed to parse functions response: %w", err)
	}

	if functions, ok := data["functions"].([]interface{}); ok {
		c.availableFunctions = make([]map[string]interface{}, len(functions))
		for i, fn := range functions {
			if fnMap, ok := fn.(map[string]interface{}); ok {
				c.availableFunctions[i] = fnMap
			}
		}
		log.Printf("Loaded %d MCP functions", len(c.availableFunctions))
	}

	return nil
}

// CallFunction calls a function on the MCP server
func (c *Client) CallFunction(functionName string, parameters map[string]interface{}) (map[string]interface{}, error) {
	c.mu.RLock()
	connected := c.isConnected
	c.mu.RUnlock()

	if !connected {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	payload := map[string]interface{}{
		"function":   functionName,
		"parameters": parameters,
	}

	resp, err := c.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Post(c.serverURL + "/call")

	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Function call failed: %v", err)}, nil
	}

	if resp.StatusCode() != http.StatusOK {
		return map[string]interface{}{"error": fmt.Sprintf("Function call failed with status %d: %s", resp.StatusCode(), string(resp.Body()))}, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Failed to parse response: %v", err)}, nil
	}

	log.Printf("MCP function %s called successfully", functionName)
	return result, nil
}

// GetCustomerData gets customer data from MCP server
func (c *Client) GetCustomerData(customerID string) (map[string]interface{}, error) {
	return c.CallFunction("get_customer_data", map[string]interface{}{
		"customer_id": customerID,
	})
}

// UpdateCustomerData updates customer data via MCP server
func (c *Client) UpdateCustomerData(customerID string, data map[string]interface{}) (map[string]interface{}, error) {
	return c.CallFunction("update_customer_data", map[string]interface{}{
		"customer_id": customerID,
		"data":        data,
	})
}

// SearchKnowledgeBase searches knowledge base via MCP server
func (c *Client) SearchKnowledgeBase(query string) (map[string]interface{}, error) {
	return c.CallFunction("search_knowledge_base", map[string]interface{}{
		"query": query,
	})
}

// CreateTicket creates a support ticket via MCP server
func (c *Client) CreateTicket(ticketData map[string]interface{}) (map[string]interface{}, error) {
	return c.CallFunction("create_ticket", ticketData)
}

// GetAvailableFunctions returns list of available MCP functions
func (c *Client) GetAvailableFunctions() []map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.availableFunctions
}

// GetFunctionSchema gets schema for a specific function
func (c *Client) GetFunctionSchema(functionName string) map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, fn := range c.availableFunctions {
		if name, ok := fn["name"].(string); ok && name == functionName {
			return fn
		}
	}
	return nil
}

// Disconnect disconnects from MCP server
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isConnected = false
	log.Println("Disconnected from MCP server")
	return nil
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// HealthCheck checks if the MCP server is healthy
func (c *Client) HealthCheck() error {
	resp, err := c.httpClient.R().
		Get(c.serverURL + "/health")

	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode())
	}

	return nil
}

// CallFunctionWithTimeout calls a function with a custom timeout
func (c *Client) CallFunctionWithTimeout(functionName string, parameters map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	payload := map[string]interface{}{
		"function":   functionName,
		"parameters": parameters,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Failed to marshal payload: %v", err)}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.serverURL+"/call", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Failed to create request: %v", err)}, nil
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Request failed: %v", err)}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Failed to read response: %v", err)}, nil
	}

	if resp.StatusCode() != http.StatusOK {
		return map[string]interface{}{"error": fmt.Sprintf("Function call failed with status %d: %s", resp.StatusCode(), string(body))}, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Failed to parse response: %v", err)}, nil
	}

	return result, nil
}
