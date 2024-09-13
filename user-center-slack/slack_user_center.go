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
	"embed"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/apache/incubator-answer-plugins/util"

	"github.com/apache/incubator-answer-plugins/user-center-slack/i18n"
	"github.com/apache/incubator-answer/plugin"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/segmentfault/pacman/log"
)

//go:embed  info.yaml
var Info embed.FS

type UserCenter struct {
	Config          *UserCenterConfig
	SlackClient     *SlackClient // 替换为 Slack 的客户端或相关工具
	UserConfigCache *UserConfigCache
	Cache           *cache.Cache
	syncLock        sync.Mutex
	syncing         bool
	syncSuccess     bool
	syncTime        time.Time
}

func (uc *UserCenter) RegisterUnAuthRouter(r *gin.RouterGroup) {
	r.GET("/slack/login/url", uc.GetSlackRedirectURL)      // 获取 Slack OAuth 登录 URL
	r.GET("/slack/login/callback", uc.CheckSlackUserLogin) // 处理 OAuth 回调
}

func (uc *UserCenter) RegisterAuthUserRouter(r *gin.RouterGroup) {
}

func (uc *UserCenter) RegisterAuthAdminRouter(r *gin.RouterGroup) {
	r.GET("/slack/sync", uc.Sync) // 添加同步 Slack 用户功能
	r.GET("/slack/data", uc.Data) // 获取 Slack 用户数据
}

func (uc *UserCenter) AfterLogin(externalID, accessToken string) {
	log.Debugf("user %s is login", externalID)
	uc.Cache.Set(externalID, accessToken, time.Minute*5)
}

func (uc *UserCenter) UserStatus(externalID string) (userStatus plugin.UserStatus) {
	if len(externalID) == 0 {
		return plugin.UserStatusAvailable
	}

	// 获取用户的详细信息
	userDetailInfo, err := uc.SlackClient.GetUserDetailInfo(externalID) // 改为使用Slack API
	if err != nil || userDetailInfo == nil {
		log.Errorf("Failed to get Slack user detail info: %v", err)
		return plugin.UserStatusDeleted
	}

	// 处理Slack API返回的用户状态
	if userDetailInfo.Deleted {
		return plugin.UserStatusDeleted
	}
	if !userDetailInfo.IsActive {
		return plugin.UserStatusSuspended
	}

	// 用户正常
	return plugin.UserStatusAvailable
}

func init() {
	// 初始化UserCenter，移除Company逻辑
	uc := &UserCenter{
		Config:          &UserCenterConfig{},                      // 配置项
		UserConfigCache: NewUserConfigCache(),                     // 用户配置缓存
		Cache:           cache.New(5*time.Minute, 10*time.Minute), // 缓存
		syncLock:        sync.Mutex{},                             // 同步锁
		SlackClient:     NewSlackClient(),                         // 初始化Slack API 客户端
	}

	// 注册插件
	plugin.Register(uc)

	// 开启定时任务来同步Slack用户数据
	uc.CronSyncData()
}

func (uc *UserCenter) Info() plugin.Info {
	info := &util.Info{}
	info.GetInfo(Info)

	return plugin.Info{
		Name:        plugin.MakeTranslator("Slack User Center"),
		SlugName:    "slack-user-center",
		Description: plugin.MakeTranslator("A plugin for integrating Slack user management"),
		Author:      "AnanChen",
		Version:     "1.0.0",
		Link:        "https://github.com/SeddonShen/incubator-answer-plugins/user-center-slack",
	}
}

func (uc *UserCenter) Description() plugin.UserCenterDesc {
	redirectURL := "/user-center/auth"
	desc := plugin.UserCenterDesc{
		Name:        "Slack",
		DisplayName: plugin.MakeTranslator(i18n.InfoName),
		//TODO
		Icon:                      "",
		Url:                       "",
		LoginRedirectURL:          redirectURL,
		SignUpRedirectURL:         redirectURL,
		RankAgentEnabled:          false,
		UserStatusAgentEnabled:    false,
		UserRoleAgentEnabled:      false,
		MustAuthEmailEnabled:      true,
		EnabledOriginalUserSystem: true,
	}
	return desc
}

func (uc *UserCenter) ControlCenterItems() []plugin.ControlCenter {
	var controlCenterItems []plugin.ControlCenter
	return controlCenterItems
}

func (uc *UserCenter) LoginCallback(ctx *plugin.GinContext) (userInfo *plugin.UserCenterBasicUserInfo, err error) {
	code := ctx.Query("code")
	if len(code) == 0 {
		return nil, fmt.Errorf("code is empty")
	}
	state := ctx.Query("state")
	if len(state) == 0 {
		return nil, fmt.Errorf("state is empty")
	}
	log.Debugf("request code: %s, state: %s", code, state)

	// 替换为 Slack OAuth 授权码获取用户信息的流程
	info, err := uc.SlackClient.ExchangeCodeForUser(code)
	if err != nil {
		return nil, fmt.Errorf("auth user failed: %w", err)
	}

	if !info.IsAvailable {
		return nil, fmt.Errorf("user is not available")
	}
	if len(info.Email) == 0 {
		ctx.Redirect(http.StatusFound, "/user-center/auth-failed")
		ctx.Abort()
		return nil, fmt.Errorf("user email is empty")
	}

	// 构建用户信息
	userInfo = &plugin.UserCenterBasicUserInfo{
		ExternalID:  info.ID,
		Username:    info.ID,
		DisplayName: info.Name,
		Email:       info.Email,
		Rank:        0,
		Avatar:      info.Avatar,
	}

	// 将用户信息缓存
	uc.Cache.Set(state, userInfo.ExternalID, time.Minute*5)
	return userInfo, nil
}

func (uc *UserCenter) SignUpCallback(ctx *plugin.GinContext) (userInfo *plugin.UserCenterBasicUserInfo, err error) {
	return uc.LoginCallback(ctx)
}

func (uc *UserCenter) UserInfo(externalID string) (userInfo *plugin.UserCenterBasicUserInfo, err error) {
	// 使用 Slack API 获取用户详细信息
	userDetailInfo, err := uc.SlackClient.GetUserDetailInfo(externalID)
	if err != nil {
		log.Errorf("get Slack user detail info failed: %v", err)
		userInfo = &plugin.UserCenterBasicUserInfo{
			ExternalID: externalID,
			Status:     plugin.UserStatusDeleted,
		}
		return userInfo, nil
	}

	// 构建用户信息
	userInfo = &plugin.UserCenterBasicUserInfo{
		ExternalID:  userDetailInfo.ID,
		Username:    userDetailInfo.ID,
		DisplayName: userDetailInfo.Name,
		Bio:         fmt.Sprintf("Slack user: %s", userDetailInfo.Name),
	}

	// 根据 Slack 的用户状态设置状态
	if userDetailInfo.Deleted {
		userInfo.Status = plugin.UserStatusDeleted
	} else {
		userInfo.Status = plugin.UserStatusAvailable
	}
	return userInfo, nil
}

func (uc *UserCenter) UserList(externalIDs []string) (userList []*plugin.UserCenterBasicUserInfo, err error) {
	userList = make([]*plugin.UserCenterBasicUserInfo, 0)
	return userList, nil
}

func (uc *UserCenter) UserSettings(externalID string) (userSettings *plugin.SettingInfo, err error) {
	return &plugin.SettingInfo{
		ProfileSettingRedirectURL: "",
		AccountSettingRedirectURL: "",
	}, nil
}

func (uc *UserCenter) PersonalBranding(externalID string) (branding []*plugin.PersonalBranding) {
	return branding
}
