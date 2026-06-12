package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	clientHeader = "X-Zuzunza-Decoder-Client"
	clientValue  = "haedoekgi/1.0"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (c *Client) newRequest(method, path string) (*http.Request, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Origin", "haedoekgi")
	req.Header.Set(clientHeader, clientValue)
	return req, nil
}

type SearchResponse struct {
	Items  []SearchItem `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type SearchItem struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	MakerID          string `json:"makerId"`
	MakerDisplayName string `json:"makerDisplayName"`
	MakerNickname    string `json:"makerNickname"`
	CreatedAt        string `json:"createdAt"`
}

func (c *Client) Search(q, title, makerID, nickname string, limit, offset int) (*SearchResponse, error) {
	vals := url.Values{}
	if q != "" {
		vals.Set("q", q)
	}
	if title != "" {
		vals.Set("title", title)
	}
	if makerID != "" {
		vals.Set("maker_id", makerID)
	}
	if nickname != "" {
		vals.Set("nickname", nickname)
	}
	if limit > 0 {
		vals.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		vals.Set("offset", strconv.Itoa(offset))
	}
	req, err := c.newRequest(http.MethodGet, "/api/decoder/v1/search?"+vals.Encode())
	if err != nil {
		return nil, err
	}
	return decodeJSON[SearchResponse](c, req)
}

type QuotaResponse struct {
	DownloadCount    int  `json:"downloadCount"`
	DownloadLimit    int  `json:"downloadLimit"`
	DecryptCount     int  `json:"decryptCount"`
	DecryptLimit     int  `json:"decryptLimit"`
	BandwidthLimited bool `json:"bandwidthLimited"`
}

func (c *Client) Quota() (*QuotaResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/decoder/v1/quota")
	if err != nil {
		return nil, err
	}
	return decodeJSON[QuotaResponse](c, req)
}

func (c *Client) Download(id, destPath string, onProgress func(written, total int64, limited bool)) error {
	req, err := c.newRequest(http.MethodGet, "/api/decoder/v1/flash/"+url.PathEscape(id)+"/download")
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkResponse(resp); err != nil {
		return err
	}
	limited := strings.EqualFold(resp.Header.Get("X-Decoder-Bandwidth-Limited"), "true")
	return saveBody(resp.Body, destPath, resp.ContentLength, limited, onProgress)
}

func (c *Client) Decrypt(id, destPath string) error {
	req, err := c.newRequest(http.MethodGet, "/api/decoder/v1/flash/"+url.PathEscape(id)+"/decrypt")
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkResponse(resp); err != nil {
		return err
	}
	return saveBody(resp.Body, destPath, resp.ContentLength, false, nil)
}

func decodeJSON[T any](c *Client, req *http.Request) (*T, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	msg := strings.TrimSpace(string(body))
	switch resp.StatusCode {
	case http.StatusForbidden:
		return fmt.Errorf("접근 거부: 해독기 클라이언트 헤더가 필요합니다")
	case http.StatusTooManyRequests:
		retry := resp.Header.Get("Retry-After")
		remain := resp.Header.Get("X-Decoder-Quota-Remaining")
		return fmt.Errorf("일일 해독 할당량 초과 (잔여 %s, Retry-After %s초)", remain, retry)
	default:
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("API 오류 (%d): %s", resp.StatusCode, msg)
	}
}

func saveBody(r io.Reader, destPath string, total int64, limited bool, onProgress func(written, total int64, limited bool)) error {
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	written, err := copyWithProgress(f, r, total, limited, onProgress)
	if err != nil {
		return err
	}
	if onProgress != nil {
		onProgress(written, total, limited)
	}
	return nil
}

func copyWithProgress(dst io.Writer, src io.Reader, total int64, limited bool, onProgress func(written, total int64, limited bool)) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	last := time.Now()
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if onProgress != nil && time.Since(last) > 500*time.Millisecond {
				onProgress(written, total, limited)
				last = time.Now()
			}
		}
		if er != nil {
			if er == io.EOF {
				return written, nil
			}
			return written, er
		}
	}
}

func FormatQuota(q *QuotaResponse) string {
	limited := "아니오"
	if q.BandwidthLimited {
		limited = "예 (1Mbps 제한)"
	}
	return fmt.Sprintf("다운로드 %d/%d, 해독 %d/%d, 대역폭 제한: %s",
		q.DownloadCount, q.DownloadLimit, q.DecryptCount, q.DecryptLimit, limited)
}
