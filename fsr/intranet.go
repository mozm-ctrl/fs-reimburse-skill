package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Talks to the nodelink-intranet PocketBase backend exactly the way its web frontend does.

const defaultURL = "http://10.1.2.247"

var httpClient = &http.Client{Timeout: 60 * time.Second}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Auth struct {
	Token  string `json:"token"`
	Record User   `json:"record"`
}

type Record struct {
	ID                  string  `json:"id"`
	Date                string  `json:"date"`
	Type                string  `json:"type"`
	Status              string  `json:"status"`
	ReimbursementAmount float64 `json:"reimbursementAmount"`
	Memo                string  `json:"memo"`
}

// errExpired means the site rejected the stored token; the user has to log in again.
var errExpired = fmt.Errorf("登录已过期，请在自己的终端重新运行：fsr login")

func postJSON(base, path, token string, body any) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest("POST", base+path, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连不上报销网站 %s（需要在公司内网或 VPN 下）: %w", base, err)
	}
	return res, nil
}

// apiError extracts PocketBase's {"message": ..., "data": {...}} so the site's own rule text
// (for example the daily check-in requirement) reaches the user.
func apiError(res *http.Response) error {
	raw, _ := io.ReadAll(res.Body)
	var pb struct {
		Message string                     `json:"message"`
		Data    map[string]json.RawMessage `json:"data"`
	}
	msg := strings.TrimSpace(string(raw))
	if json.Unmarshal(raw, &pb) == nil && pb.Message != "" {
		msg = pb.Message
		for k, v := range pb.Data {
			msg += fmt.Sprintf("; %s: %s", k, v)
		}
	}
	return fmt.Errorf("报销网站返回 %d: %s", res.StatusCode, msg)
}

func Login(base, email, password string) (*Auth, error) {
	res, err := postJSON(base, "/api/collections/users/auth-with-password", "", map[string]string{"identity": email, "password": password})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == 400 {
		return nil, fmt.Errorf("登录失败：网站拒绝了邮箱 %s 和这个密码", email)
	}
	if res.StatusCode != 200 {
		return nil, apiError(res)
	}
	var a Auth
	return &a, json.NewDecoder(res.Body).Decode(&a)
}

// Refresh extends the token's lifetime and returns the new token.
func Refresh(base, token string) (*Auth, error) {
	res, err := postJSON(base, "/api/collections/users/auth-refresh", token, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == 401 || res.StatusCode == 403 || res.StatusCode == 404 {
		return nil, errExpired
	}
	if res.StatusCode != 200 {
		return nil, apiError(res)
	}
	var a Auth
	return &a, json.NewDecoder(res.Body).Decode(&a)
}

func CreateRecord(base string, a *Auth, dataURLs []string, typ, memo, date string) (*Record, error) {
	receipts, _ := json.Marshal(dataURLs)
	body := map[string]any{
		"userId": a.Record.ID, "userEmail": a.Record.Email, "userName": a.Record.Name,
		"date": date, "type": typ, "status": "pending", "reimbursementAmount": 0,
		"receiptUrl": string(receipts),
	}
	if typ == "SHOPPING_RECEIPT" && memo != "" {
		body["memo"] = memo
	}
	res, err := postJSON(base, "/api/collections/records/records", a.Token, body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, apiError(res)
	}
	var rec Record
	return &rec, json.NewDecoder(res.Body).Decode(&rec)
}

func ListRecords(base string, a *Auth, n int) ([]Record, error) {
	q := url.Values{"filter": {fmt.Sprintf(`userId="%s"`, a.Record.ID)}, "sort": {"-created"}, "perPage": {fmt.Sprint(n)}}
	req, err := http.NewRequest("GET", base+"/api/collections/records/records?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", a.Token)
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连不上报销网站 %s: %w", base, err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, apiError(res)
	}
	var out struct{ Items []Record }
	return out.Items, json.NewDecoder(res.Body).Decode(&out)
}
