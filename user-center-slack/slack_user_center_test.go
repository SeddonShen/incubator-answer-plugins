package slack_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	slack "github.com/apache/incubator-answer-plugins/user-center-slack"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestGetSlackRedirectURL(t *testing.T) {
	// 创建测试 Gin 上下文
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 创建 UserCenter 实例
	uc := &slack.UserCenter{
		// 初始化需要的字段
		Cache: cache.New(5*time.Minute, 10*time.Minute),
	}

	// 执行函数
	uc.GetSlackRedirectURL(c)

	// 检查响应
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "redirect_url") // 检查是否包含正确的字段

	fmt.Println(w.Body.String())
	fmt.Print("TestGetSlackRedirectURL Finished")
}

func TestCheckSlackUserLogin(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	uc := &slack.UserCenter{
		Cache: cache.New(5*time.Minute, 10*time.Minute),
	}

	// 设置缓存中的 state 和 external_id
	state := "teststate"
	externalID := "U12345"
	uc.Cache.Set(state, externalID, cache.DefaultExpiration)
	uc.Cache.Set(externalID, "testtoken", cache.DefaultExpiration)

	c.Request = httptest.NewRequest("GET", "/?state=teststate", nil)

	// 执行函数
	uc.CheckSlackUserLogin(c)

	// 检查响应
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "is_login")
	assert.Contains(t, w.Body.String(), "testtoken")

	fmt.Println(w.Body.String())
	fmt.Print("TestCheckSlackUserLogin Finished")
}

// func TestSync(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	// 正确调用 NewMockISlackClient
// 	mockSlackClient := slack.NewMockISlackClient(ctrl)

// 	mockSlackClient.EXPECT().ListUsers().Return([]slack.SlackUserDetail{
// 		{ID: "U12345", Name: "Test User", IsActive: true},
// 	}, nil)

// 	uc := &slack.UserCenter{
// 		SlackClient: mockSlackClient,
// 		Cache:       cache.New(5*time.Minute, 10*time.Minute),
// 	}

// 	w := httptest.NewRecorder()
// 	c, _ := gin.CreateTestContext(w)

// 	uc.Sync(c)

// 	assert.Equal(t, http.StatusOK, w.Code)
// 	assert.Contains(t, w.Body.String(), "User data synced successfully")
// }

// func TestGetUserInfo(t *testing.T) {
// 	// 模拟 Slack API 返回用户信息
// 	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusOK)
// 		w.Header().Set("Content-Type", "application/json")
// 		fmt.Fprintln(w, `{"ok": true, "user": {"id": "U12345", "name": "Test User"}}`)
// 	}))
// 	defer mockServer.Close()

// 	// 使用 mockServer.URL 来替代 Slack API URL 进行测试
// 	userInfo, err := GetUserInfo("mock_access_token", "U12345")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if userInfo.ID != "U12345" {
// 		t.Errorf("Expected user ID to be 'U12345', got %s", userInfo.ID)
// 	}
// }
