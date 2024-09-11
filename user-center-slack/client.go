package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"io/ioutil"
	"github.com/segmentfault/pacman/log"
)

// SlackClient 用于与 Slack API 进行交互
type SlackClient struct {
	AccessToken string
	ClientID    string
	ClientSecret string
	RedirectURI  string
}

// NewSlackClient 创建一个新的 SlackClient
func NewSlackClient() *SlackClient {
	return &SlackClient{
		ClientID:     "your-slack-client-id",  // 从配置文件或环境变量中加载
		ClientSecret: "your-slack-client-secret",
		RedirectURI:  "your-slack-redirect-uri",
	}
}

// SlackUserDetail 是 Slack 用户详细信息的结构体
type SlackUserDetail struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Deleted bool   `json:"deleted"`
	IsActive bool  `json:"is_active"`
	Profile struct {
		Email string `json:"email"`
	} `json:"profile"`
}

// SlackUserInfo 是 OAuth 授权后的用户信息
type SlackUserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"image_192"`
	IsAvailable bool
}

// ExchangeCodeForUser 通过 OAuth 授权码获取 Slack 用户访问令牌和用户信息
func (sc *SlackClient) ExchangeCodeForUser(code string) (*SlackUserInfo, error) {
	// 设置 OAuth 交换令牌请求的 URL
	data := url.Values{}
	data.Set("client_id", sc.ClientID)
	data.Set("client_secret", sc.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", sc.RedirectURI)

	resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", data)
	if err != nil {
		log.Errorf("Failed to exchange code for token: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %v", err)
	}

	// 解析 OAuth 交换后的返回信息
	var tokenResp struct {
		AccessToken string      `json:"access_token"`
		AuthedUser  SlackUserInfo `json:"authed_user"`
		OK          bool        `json:"ok"`
		Error       string      `json:"error"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("Failed to parse token response: %v", err)
	}

	if !tokenResp.OK {
		return nil, fmt.Errorf("Slack API error: %s", tokenResp.Error)
	}

	sc.AccessToken = tokenResp.AccessToken
	return &tokenResp.AuthedUser, nil
}

// GetUserDetailInfo 使用 Slack API 获取指定用户的详细信息
func (sc *SlackClient) GetUserDetailInfo(userID string) (*SlackUserDetail, error) {
	url := fmt.Sprintf("https://slack.com/api/users.info?user=%s", userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %v", err)
	}

	// 设置授权头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.AccessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %v", err)
	}

	var result struct {
		OK    bool            `json:"ok"`
		User  SlackUserDetail `json:"user"`
		Error string          `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("Failed to parse user detail response: %v", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("Slack API error: %s", result.Error)
	}

	return &result.User, nil
}

// ListUsers 获取工作区中的所有用户列表
func (sc *SlackClient) ListUsers() ([]SlackUserDetail, error) {
	url := "https://slack.com/api/users.list"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %v", err)
	}

	// 设置授权头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.AccessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %v", err)
	}

	var result struct {
		OK      bool              `json:"ok"`
		Members []SlackUserDetail `json:"members"`
		Error   string            `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("Failed to parse users list response: %v", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("Slack API error: %s", result.Error)
	}

	return result.Members, nil
}

