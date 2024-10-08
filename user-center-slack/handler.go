/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package slack

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/apache/incubator-answer/plugin"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/segmentfault/pacman/log"
)

// RespBody response body.
type RespBody struct {
	// http code
	Code int `json:"code"`
	// reason key
	Reason string `json:"reason"`
	// response message
	Message string `json:"msg"`
	// response data
	Data interface{} `json:"data"`
}

// NewRespBodyData new response body with data
func NewRespBodyData(code int, reason string, data interface{}) *RespBody {
	return &RespBody{
		Code:   code,
		Reason: reason,
		Data:   data,
	}
}

// func (uc *UserCenter) GetRedirectURL(ctx *gin.Context) {
// 	authorizeUrl := fmt.Sprintf("%s/answer/api/v1/user-center/login/callback", plugin.SiteURL())
// 	redirectURL := uc.Company.GetRedirectURL(authorizeUrl)
// 	state := genNonce()
// 	redirectURL = strings.ReplaceAll(redirectURL, "STATE", state)
// 	ctx.JSON(http.StatusOK, NewRespBodyData(http.StatusOK, "success", map[string]string{
// 		"redirect_url": redirectURL,
// 		"key":          state,
// 	}))
// }

func (uc *UserCenter) GetSlackRedirectURL(ctx *gin.Context) {
	// 定义 Slack OAuth 2.0 的相关信息
	clientID := "7420840065700.7709579732657" // 替换为你的 Slack Client ID
	// redirectURI := "https://as.0vo.lol//slack/login/callback"  // 你的回调地址
	redirectURI := fmt.Sprintf("%s/answer/api/v1/user-center/login/callback", plugin.SiteURL())
	scope := "chat:write,commands,groups:write,im:write,incoming-webhook,mpim:write,users:read,users:read.emai" // 需要的权限范围
	state := genNonce()                                                                                         // 生成防止CSRF攻击的state值

	// 构建 Slack OAuth 2.0 的授权 URL
	redirectURL := fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&scope=%s&redirect_uri=%s&state=%s",
		clientID, scope, redirectURI, state,
	)

	// 返回 JSON 响应，包括重定向 URL 和 state
	ctx.JSON(http.StatusOK, NewRespBodyData(http.StatusOK, "success", map[string]string{
		"redirect_url": redirectURL,
		"key":          state,
	}))
}

func (uc *UserCenter) CheckSlackUserLogin(ctx *gin.Context) {
	// 从请求中获取 state 参数
	state := ctx.Query("state")

	// 检查缓存中是否存在该 state 对应的登录信息
	val, exist := uc.Cache.Get(state)
	if !exist {
		// 如果不存在对应的登录信息，返回未登录状态
		ctx.JSON(http.StatusOK, NewRespBodyData(http.StatusOK, "success", map[string]any{
			"is_login": false,
			"token":    "",
		}))
		return
	}

	// 获取授权信息中的 external_id（通常是用户的Slack ID或其他标识）
	token := ""
	externalID, _ := val.(string)               // 假设缓存中存的是用户的 external_id
	tokenStr, exist := uc.Cache.Get(externalID) // 根据 external_id 获取缓存中的 token
	if exist {
		// 如果缓存中存在该 external_id 的 token，则获取 token
		token, _ = tokenStr.(string)
	}

	// 返回登录状态和 token（如果有）
	ctx.JSON(http.StatusOK, NewRespBodyData(http.StatusOK, "success", map[string]any{
		"is_login": len(token) > 0,
		"token":    token,
	}))
}

func (uc *UserCenter) Sync(ctx *gin.Context) {
	// 调用Slack API获取工作区用户信息
	uc.syncSlackUsers()

	if uc.syncSuccess {
		ctx.JSON(http.StatusOK, NewRespBodyData(http.StatusOK, "success", map[string]any{
			"message": "User data synced successfully",
		}))
		return
	}

	errRespBodyData := NewRespBodyData(http.StatusBadRequest, "error", map[string]any{
		"err_type": "toast",
	})
	errRespBodyData.Message = "Failed to sync user data"
	ctx.JSON(http.StatusBadRequest, errRespBodyData)
}

func (uc *UserCenter) syncSlackUsers() {
	// 使用 Slack API 获取用户列表
	users, err := uc.SlackClient.ListUsers()
	if err != nil {
		log.Errorf("Failed to sync Slack users: %v", err)
		uc.syncSuccess = false
		return
	}

	// 将同步后的用户信息存储到缓存
	// 假设将用户列表存储在缓存的 "slack_users" 键下，缓存的过期时间根据你的需求调整
	uc.Cache.Set("slack_users", users, cache.DefaultExpiration)

	uc.syncSuccess = true
}

func (uc *UserCenter) Data(ctx *gin.Context) {
	// 从缓存中获取用户信息
	users, found := uc.Cache.Get("slack_users")
	if !found {
		ctx.JSON(http.StatusInternalServerError, NewRespBodyData(http.StatusInternalServerError, "error", map[string]any{
			"message": "No user data available",
		}))
		return
	}

	// 可以通过Slack API获取工作区的详细信息，这里为简单起见，直接返回静态数据
	workspaceInfo := map[string]string{
		"name":   "My Workspace", // 可通过 Slack API 动态获取
		"id":     "T1234567890",  // 可通过 Slack API 动态获取
		"domain": "myworkspace",  // 可通过 Slack API 动态获取
	}

	// 返回用户和工作区信息
	ctx.JSON(http.StatusOK, NewRespBodyData(http.StatusOK, "success", map[string]any{
		"users":     users,
		"workspace": workspaceInfo,
	}))
}

// 随机生成 nonce
func genNonce() string {
	bytes := make([]byte, 10)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
