package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const salesforceAPIVersion = "v59.0"

// SalesforceClient reads Salesforce CRM data through SOQL query endpoints.
type SalesforceClient struct {
	client *http.Client
}

// NewSalesforceClient creates a Salesforce API client.
func NewSalesforceClient() *SalesforceClient {
	return &SalesforceClient{client: http.DefaultClient}
}

// SalesforceAccount is the subset of Account fields used by PulseScore.
type SalesforceAccount struct {
	ID               string    `json:"Id"`
	Name             string    `json:"Name"`
	Website          string    `json:"Website"`
	Industry         string    `json:"Industry"`
	NumberOfEmployees int      `json:"NumberOfEmployees"`
	AnnualRevenue    float64   `json:"AnnualRevenue"`
	LastModifiedDate time.Time `json:"LastModifiedDate"`
}

// SalesforceContact is the subset of Contact fields used by PulseScore.
type SalesforceContact struct {
	ID               string    `json:"Id"`
	Email            string    `json:"Email"`
	FirstName        string    `json:"FirstName"`
	LastName         string    `json:"LastName"`
	AccountID        string    `json:"AccountId"`
	LastModifiedDate time.Time `json:"LastModifiedDate"`
}

// SalesforceOpportunity is the subset of Opportunity fields used by PulseScore.
type SalesforceOpportunity struct {
	ID               string    `json:"Id"`
	Name             string    `json:"Name"`
	StageName        string    `json:"StageName"`
	Amount           float64   `json:"Amount"`
	CloseDate        string    `json:"CloseDate"`
	AccountID        string    `json:"AccountId"`
	LastModifiedDate time.Time `json:"LastModifiedDate"`
}

// ListAccounts fetches Salesforce accounts.
func (c *SalesforceClient) ListAccounts(ctx context.Context, access SalesforceAccess, since *time.Time) ([]SalesforceAccount, error) {
	query := "SELECT Id, Name, Website, Industry, NumberOfEmployees, AnnualRevenue, LastModifiedDate FROM Account"
	if since != nil {
		query += " WHERE LastModifiedDate >= " + formatSalesforceTime(*since)
	}
	return querySalesforce[SalesforceAccount](ctx, c.client, access, query)
}

// ListContacts fetches Salesforce contacts that can map to customers by email.
func (c *SalesforceClient) ListContacts(ctx context.Context, access SalesforceAccess, since *time.Time) ([]SalesforceContact, error) {
	query := "SELECT Id, Email, FirstName, LastName, AccountId, LastModifiedDate FROM Contact WHERE Email != null"
	if since != nil {
		query += " AND LastModifiedDate >= " + formatSalesforceTime(*since)
	}
	return querySalesforce[SalesforceContact](ctx, c.client, access, query)
}

// ListOpportunities fetches Salesforce opportunities for customer timeline events.
func (c *SalesforceClient) ListOpportunities(ctx context.Context, access SalesforceAccess, since *time.Time) ([]SalesforceOpportunity, error) {
	query := "SELECT Id, Name, StageName, Amount, CloseDate, AccountId, LastModifiedDate FROM Opportunity"
	if since != nil {
		query += " WHERE LastModifiedDate >= " + formatSalesforceTime(*since)
	}
	return querySalesforce[SalesforceOpportunity](ctx, c.client, access, query)
}

type salesforceQueryResponse[T any] struct {
	Done           bool   `json:"done"`
	NextRecordsURL string `json:"nextRecordsUrl"`
	Records        []T    `json:"records"`
}

func querySalesforce[T any](ctx context.Context, client *http.Client, access SalesforceAccess, soql string) ([]T, error) {
	if client == nil {
		client = http.DefaultClient
	}
	nextURL := strings.TrimRight(access.InstanceURL, "/") + "/services/data/" + salesforceAPIVersion + "/query?q=" + url.QueryEscape(soql)
	var all []T
	for nextURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+access.AccessToken)
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("salesforce request: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read response: %w", readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close response: %w", closeErr)
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("salesforce query failed with status %d: %s", resp.StatusCode, string(body))
		}
		var page salesforceQueryResponse[T]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		all = append(all, page.Records...)
		if page.Done || page.NextRecordsURL == "" {
			nextURL = ""
			continue
		}
		if strings.HasPrefix(page.NextRecordsURL, "http") {
			nextURL = page.NextRecordsURL
		} else {
			nextURL = strings.TrimRight(access.InstanceURL, "/") + page.NextRecordsURL
		}
	}
	return all, nil
}

func formatSalesforceTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
