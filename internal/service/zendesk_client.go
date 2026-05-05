package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// ZendeskClient provides rate-limited access to Zendesk Support API v2.
type ZendeskClient struct {
	client  *http.Client
	limiter *rate.Limiter
}

// NewZendeskClient creates a new ZendeskClient.
func NewZendeskClient() *ZendeskClient {
	return &ZendeskClient{client: &http.Client{}, limiter: rate.NewLimiter(rate.Limit(10), 10)}
}

// ZendeskUserListResponse represents Zendesk user list response.
type ZendeskUserListResponse struct {
	Users       []ZendeskAPIUser `json:"users"`
	NextPage    string           `json:"next_page"`
	Count       int              `json:"count"`
	EndOfStream bool             `json:"end_of_stream"`
}

// ZendeskAPIUser is a Zendesk end-user.
type ZendeskAPIUser struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ZendeskTicketListResponse represents Zendesk ticket list response.
type ZendeskTicketListResponse struct {
	Tickets     []ZendeskAPITicket `json:"tickets"`
	NextPage    string             `json:"next_page"`
	Count       int                `json:"count"`
	EndOfStream bool               `json:"end_of_stream"`
}

// ZendeskAPITicket is a Zendesk support ticket.
type ZendeskAPITicket struct {
	ID          int64      `json:"id"`
	RequesterID int64      `json:"requester_id"`
	SubmitterID int64      `json:"submitter_id"`
	Subject     string     `json:"subject"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	Type        string     `json:"type"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SolvedAt    *time.Time `json:"solved_at"`
}

// ListUsers fetches Zendesk end-users.
func (c *ZendeskClient) ListUsers(ctx context.Context, accessToken, subdomain, pageURL string) (*ZendeskUserListResponse, error) {
	endpoint := pageURL
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.zendesk.com/api/v2/users.json?role[]=end-user&page[size]=100", subdomain)
	}
	return zendeskGet[ZendeskUserListResponse](ctx, c, endpoint, accessToken)
}

// ListUsersUpdatedSince fetches Zendesk users updated since a timestamp.
func (c *ZendeskClient) ListUsersUpdatedSince(ctx context.Context, accessToken, subdomain string, since time.Time, pageURL string) (*ZendeskUserListResponse, error) {
	endpoint := pageURL
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.zendesk.com/api/v2/incremental/users.json?start_time=%d", subdomain, since.Unix())
	}
	return zendeskGet[ZendeskUserListResponse](ctx, c, endpoint, accessToken)
}

// ListTickets fetches Zendesk tickets.
func (c *ZendeskClient) ListTickets(ctx context.Context, accessToken, subdomain, pageURL string) (*ZendeskTicketListResponse, error) {
	endpoint := pageURL
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.zendesk.com/api/v2/tickets.json?page[size]=100", subdomain)
	}
	return zendeskGet[ZendeskTicketListResponse](ctx, c, endpoint, accessToken)
}

// ListTicketsUpdatedSince fetches Zendesk tickets updated since a timestamp.
func (c *ZendeskClient) ListTicketsUpdatedSince(ctx context.Context, accessToken, subdomain string, since time.Time, pageURL string) (*ZendeskTicketListResponse, error) {
	endpoint := pageURL
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.zendesk.com/api/v2/incremental/tickets.json?start_time=%d", subdomain, since.Unix())
	}
	return zendeskGet[ZendeskTicketListResponse](ctx, c, endpoint, accessToken)
}

func zendeskGet[T any](ctx context.Context, c *ZendeskClient, endpoint, accessToken string) (*T, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zendesk api error: status %d, body: %s", resp.StatusCode, string(body))
	}
	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &result, nil
}

func zendeskInt64ID(id int64) string { return strconv.FormatInt(id, 10) }
