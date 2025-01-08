package frollo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const ClientID = "UCyPI63qO8fVsjnxNcEuVbHDOWSr8tQiDTrFsrb93o0"

type Client struct {
	accessToken string
}

func NewClient() *Client {
	return &Client{
		accessToken: "",
	}
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int    `json:"created_at"`
	IDToken      string `json:"id_token"`
}

type Provider struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	SmallLogoURL string `json:"small_logo_url"`
	LargeLogoURL string `json:"large_logo_url"`
}

type AccountAttributes struct {
	Container      string `json:"container"`
	AccountType    string `json:"account_type"`
	Group          string `json:"group"`
	Classification string `json:"classification"`
}

type Balance struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type RefreshStatus struct {
	Status        string `json:"status"`
	SubStatus     string `json:"sub_status"`
	LastRefreshed string `json:"last_refreshed"`
	NextRefresh   string `json:"next_refresh"`
}

type FrolloAccount struct {
	ID                int                `json:"id"`
	Aggregator        string             `json:"aggregator"`
	ExternalID        string             `json:"external_id"`
	ProviderAccountID int                `json:"provider_account_id"`
	Provider          *Provider          `json:"provider"`
	AccountName       string             `json:"account_name"`
	AccountStatus     string             `json:"account_status"`
	JointAccount      bool               `json:"joint_account"`
	OwnerType         string             `json:"owner_type"`
	AccountAttributes *AccountAttributes `json:"account_attributes"`
	Included          bool               `json:"included"`
	Favourite         bool               `json:"favourite"`
	Hidden            bool               `json:"hidden"`
	PrimaryBalance    *Balance           `json:"primary_balance"`
	CurrentBalance    *Balance           `json:"current_balance"`
	RefreshStatus     *RefreshStatus     `json:"refresh_status"`
	ProductsAvailable bool               `json:"products_available"`
	Asset             bool               `json:"asset"`
	PayIDs            []interface{}      `json:"payids"`
}

type Description struct {
	Original string `json:"original"`
	Simple   string `json:"simple"`
}

type Image struct {
	ID            int    `json:"id"`
	SmallImageURL string `json:"small_image_url"`
	LargeImageURL string `json:"large_image_url"`
}

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Colour      string `json:"colour"`
	Image       Image  `json:"image"`
}

type Merchant struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	MerchantType string `json:"merchant_type"`
	ImageURL     string `json:"image_url"`
	Phone        string `json:"phone"`
	Website      string `json:"website"`
}

type FrolloTransaction struct {
	ID                int           `json:"id"`
	AccountID         int           `json:"account_id"`
	AccountAggregator string        `json:"account_aggregator"`
	BaseType          string        `json:"base_type"`
	Status            string        `json:"status"`
	Included          bool          `json:"included"`
	TransactionDate   string        `json:"transaction_date"`
	PostDate          string        `json:"post_date"`
	Amount            *Balance      `json:"amount"`
	Description       *Description  `json:"description"`
	BudgetCategory    string        `json:"budget_category"`
	Category          *Category     `json:"category"`
	Merchant          *Merchant     `json:"merchant"`
	UserTags          []interface{} `json:"user_tags"`
	Reference         string        `json:"reference"`
	Type              string        `json:"type"`
}

type Cursors struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

type Paging struct {
	Cursors *Cursors `json:"cursors"`
	Total   int      `json:"total"`
}

type GetTransactionsResponse struct {
	Data   []*FrolloTransaction `json:"data"`
	Paging Paging               `json:"paging"`
}

func (c *Client) Login(username, password string) (*LoginResponse, error) {
	url := "https://id.frollo.us/oauth/token"
	payload := map[string]string{
		"grant_type": "password",
		"domain":     "api.frollo.us",
		"client_id":  ClientID,
		"username":   username,
		"password":   password,
		"scope":      "offline_access email openid",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, err
	}

	c.accessToken = loginResp.AccessToken

	return &loginResp, nil
}

func (c *Client) GetAccounts() ([]FrolloAccount, error) {
	url := "https://api.frollo.us/api/v2/aggregation/accounts"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	setHeaders(req, c.accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var accounts []FrolloAccount
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (c *Client) GetTransactions(accountID string, fromDate, toDate time.Time) ([]*FrolloTransaction, error) {
	url := fmt.Sprintf("https://api.frollo.us/api/v2/aggregation/transactions?account_ids=%s&size=150", accountID)
	if !fromDate.IsZero() {
		url += fmt.Sprintf("&from_date=%s", fromDate.Format("2006-01-02"))
	}

	if !toDate.IsZero() {
		url += fmt.Sprintf("&to_date=%s", toDate.Format("2006-01-02"))
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	setHeaders(req, c.accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var transResp GetTransactionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&transResp); err != nil {
		return nil, err
	}

	return transResp.Data, nil
}

func (c *Client) GetAccount(accountID string) (*FrolloAccount, error) {
	url := fmt.Sprintf("https://api.frollo.us/api/v2/aggregation/accounts/%s", accountID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	setHeaders(req, c.accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var account FrolloAccount
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, err
	}

	return &account, nil
}

func (c *Client) SyncAccounts() (map[string]interface{}, error) {
	url := "https://api.frollo.us/api/v2/aggregation/provideraccounts/sync"
	req, err := http.NewRequest("POST", url, strings.NewReader(""))
	if err != nil {
		return nil, err
	}

	setHeaders(req, c.accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func setHeaders(req *http.Request, accessToken string) {
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Api-Version", "2.26")
	req.Header.Set("X-Bundle-Id", "us.frollo.frollosdk")
	req.Header.Set("X-Device-Version", "Android12")
	req.Header.Set("X-Software-Version", "SDK3.28.0-B3270|APP2.26.0-B104594")
	req.Header.Set("Host", "api.frollo.us")
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("User-Agent", "okhttp/4.12.0")
	req.Header.Set("Content-Type", "application/json")
}
