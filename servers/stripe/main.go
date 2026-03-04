// mcpfs-stripe: Stripe MCP resource server for mcpfs.
// Uses mcpserve framework. Speaks MCP JSON-RPC over stdio.
//
// Resources:
//   stripe://customers                      - recent customers
//   stripe://customers/{id}                 - customer details
//   stripe://products                       - all products
//   stripe://products/{id}                  - product details
//   stripe://prices                         - all prices
//   stripe://subscriptions                  - active subscriptions
//   stripe://subscriptions/{id}             - subscription details
//   stripe://invoices                       - recent invoices
//   stripe://invoices/{id}                  - invoice details
//   stripe://charges                        - recent charges
//   stripe://balance                        - current balance
//   stripe://events                         - recent API events
//
// Auth: STRIPE_API_KEY env var (secret key, sk_test_... or sk_live_...).
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/airshelf/mcpfs/pkg/mcpserve"
)

var apiKey string

func stripeGet(path string) (json.RawMessage, error) {
	url := "https://api.stripe.com/v1" + path
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func extractData(raw json.RawMessage) (json.RawMessage, error) {
	var list struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &list); err == nil && list.Data != nil {
		return list.Data, nil
	}
	return raw, nil
}

func readResource(uri string) (mcpserve.ReadResult, error) {
	switch {
	case uri == "stripe://customers":
		return readList("/customers?limit=50")
	case uri == "stripe://products":
		return readList("/products?limit=100&active=true")
	case uri == "stripe://prices":
		return readList("/prices?limit=100&active=true")
	case uri == "stripe://subscriptions":
		return readList("/subscriptions?limit=50&status=active")
	case uri == "stripe://invoices":
		return readList("/invoices?limit=50")
	case uri == "stripe://charges":
		return readList("/charges?limit=50")
	case uri == "stripe://balance":
		return readSingle("/balance")
	case uri == "stripe://events":
		return readList("/events?limit=50")

	case strings.HasPrefix(uri, "stripe://customers/"):
		id := strings.TrimPrefix(uri, "stripe://customers/")
		return readSingle("/customers/" + id)
	case strings.HasPrefix(uri, "stripe://products/"):
		id := strings.TrimPrefix(uri, "stripe://products/")
		return readSingle("/products/" + id)
	case strings.HasPrefix(uri, "stripe://subscriptions/"):
		id := strings.TrimPrefix(uri, "stripe://subscriptions/")
		return readSingle("/subscriptions/" + id)
	case strings.HasPrefix(uri, "stripe://invoices/"):
		id := strings.TrimPrefix(uri, "stripe://invoices/")
		return readSingle("/invoices/" + id)

	default:
		return mcpserve.ReadResult{}, fmt.Errorf("unknown resource: %s", uri)
	}
}

func readList(path string) (mcpserve.ReadResult, error) {
	raw, err := stripeGet(path)
	if err != nil {
		return mcpserve.ReadResult{}, err
	}
	data, _ := extractData(raw)
	var v interface{}
	json.Unmarshal(data, &v)
	out, _ := json.MarshalIndent(v, "", "  ")
	return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil
}

func readSingle(path string) (mcpserve.ReadResult, error) {
	raw, err := stripeGet(path)
	if err != nil {
		return mcpserve.ReadResult{}, err
	}
	var v interface{}
	json.Unmarshal(raw, &v)
	out, _ := json.MarshalIndent(v, "", "  ")
	return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil
}

func main() {
	apiKey = os.Getenv("STRIPE_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "mcpfs-stripe: STRIPE_API_KEY env var required")
		os.Exit(1)
	}

	srv := mcpserve.New("mcpfs-stripe", "0.1.0", readResource)

	srv.AddResource(mcpserve.Resource{
		URI: "stripe://customers", Name: "customers",
		Description: "Recent customers", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "stripe://products", Name: "products",
		Description: "Active products", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "stripe://prices", Name: "prices",
		Description: "Active prices", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "stripe://subscriptions", Name: "subscriptions",
		Description: "Active subscriptions", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "stripe://invoices", Name: "invoices",
		Description: "Recent invoices", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "stripe://charges", Name: "charges",
		Description: "Recent charges", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "stripe://balance", Name: "balance",
		Description: "Current account balance", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "stripe://events", Name: "events",
		Description: "Recent API events", MimeType: "application/json",
	})

	srv.AddTemplate(mcpserve.Template{
		URITemplate: "stripe://customers/{id}", Name: "customer",
		Description: "Customer details", MimeType: "application/json",
	})
	srv.AddTemplate(mcpserve.Template{
		URITemplate: "stripe://products/{id}", Name: "product",
		Description: "Product details", MimeType: "application/json",
	})
	srv.AddTemplate(mcpserve.Template{
		URITemplate: "stripe://subscriptions/{id}", Name: "subscription",
		Description: "Subscription details", MimeType: "application/json",
	})
	srv.AddTemplate(mcpserve.Template{
		URITemplate: "stripe://invoices/{id}", Name: "invoice",
		Description: "Invoice details", MimeType: "application/json",
	})

	if err := srv.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "mcpfs-stripe: %v\n", err)
		os.Exit(1)
	}
}
