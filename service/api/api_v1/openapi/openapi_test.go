package openapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sun-panel/api/api_v1/middleware"
	"sun-panel/global"
	"sun-panel/models"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type testResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func performJSONRequest(t *testing.T, router http.Handler, path, token string, body any) testResponse {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("token", token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("%s returned HTTP %d: %s", path, recorder.Code, recorder.Body.String())
	}
	response := testResponse{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode %s response: %v", path, err)
	}
	return response
}

func TestOpenAPIExtensionFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, err := gorm.Open(sqlite.Open("file:openapi_flow?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&models.User{},
		&models.OpenAPIConfig{},
		&models.ItemIconGroup{},
		&models.ItemIcon{},
	); err != nil {
		t.Fatal(err)
	}
	global.Db = database
	models.Db = database

	user := models.User{Username: "extension-user", Status: 1, Role: 2}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	handler := OpenAPI{}
	internalRouter := gin.New()
	internalRouter.Use(func(c *gin.Context) {
		c.Set("userInfo", user)
	})
	internalRouter.POST("/getConfig", handler.GetConfig)
	internalRouter.POST("/enable", handler.Enable)
	internalRouter.POST("/disable", handler.Disable)
	internalRouter.POST("/refreshToken", handler.RefreshToken)

	externalRouter := gin.New()
	externalRouter.Use(middleware.OpenAPIInterceptor)
	externalRouter.POST("/version", handler.Version)
	externalRouter.POST("/itemGroup/getList", handler.GetItemGroupList)
	externalRouter.POST("/item/create", handler.CreateItem)

	initial := performJSONRequest(t, internalRouter, "/getConfig", "", gin.H{})
	if initial.Code != 0 {
		t.Fatalf("get config failed: %+v", initial)
	}
	initialConfig := configResponse{}
	if err := json.Unmarshal(initial.Data, &initialConfig); err != nil {
		t.Fatal(err)
	}
	if initialConfig.Enabled || initialConfig.Token != "" {
		t.Fatalf("unexpected initial config: %+v", initialConfig)
	}

	enabled := performJSONRequest(t, internalRouter, "/enable", "", gin.H{})
	if enabled.Code != 0 {
		t.Fatalf("enable failed: %+v", enabled)
	}
	enabledConfig := configResponse{}
	if err := json.Unmarshal(enabled.Data, &enabledConfig); err != nil {
		t.Fatal(err)
	}
	if !enabledConfig.Enabled || enabledConfig.Token == "" {
		t.Fatalf("unexpected enabled config: %+v", enabledConfig)
	}

	invalid := performJSONRequest(t, externalRouter, "/version", "invalid-token", gin.H{})
	if invalid.Code != 1000 {
		t.Fatalf("invalid token returned code %d", invalid.Code)
	}

	version := performJSONRequest(t, externalRouter, "/version", enabledConfig.Token, gin.H{})
	if version.Code != 0 {
		t.Fatalf("version failed: %+v", version)
	}
	versionData := struct {
		Version string `json:"version"`
	}{}
	if err := json.Unmarshal(version.Data, &versionData); err != nil {
		t.Fatal(err)
	}
	if versionData.Version != "1.8.1" {
		t.Fatalf("version endpoint returned %q", versionData.Version)
	}

	groups := performJSONRequest(t, externalRouter, "/itemGroup/getList", enabledConfig.Token, gin.H{})
	if groups.Code != 0 {
		t.Fatalf("group list failed: %+v", groups)
	}
	groupData := struct {
		List []itemGroupListItem `json:"list"`
	}{}
	if err := json.Unmarshal(groups.Data, &groupData); err != nil {
		t.Fatal(err)
	}
	if len(groupData.List) != 1 || groupData.List[0].ItemGroupID == 0 {
		t.Fatalf("unexpected group response: %+v", groupData)
	}

	created := performJSONRequest(t, externalRouter, "/item/create", enabledConfig.Token, gin.H{
		"title":       "Example",
		"url":         "https://example.com",
		"lanUrl":      "http://example.lan",
		"iconUrl":     "https://example.com/favicon.ico",
		"description": "Created by extension",
		"itemGroupID": groupData.List[0].ItemGroupID,
		"isSaveIcon":  false,
	})
	if created.Code != 0 {
		t.Fatalf("create item failed: %+v", created)
	}
	item := models.ItemIcon{}
	if err := database.Where("user_id = ?", user.ID).First(&item).Error; err != nil {
		t.Fatal(err)
	}
	if item.Title != "Example" || item.ItemIconGroupId != int(groupData.List[0].ItemGroupID) {
		t.Fatalf("unexpected saved item: %+v", item)
	}

	minimal := performJSONRequest(t, externalRouter, "/item/create", enabledConfig.Token, gin.H{
		"title": "No explicit icon or group",
		"url":   "https://minimal.example/path",
	})
	if minimal.Code != 0 {
		t.Fatalf("minimal create item failed: %+v", minimal)
	}
	minimalItem := models.ItemIcon{}
	if err := database.Where("user_id = ? AND title = ?", user.ID, "No explicit icon or group").First(&minimalItem).Error; err != nil {
		t.Fatal(err)
	}
	if minimalItem.ItemIconGroupId != int(groupData.List[0].ItemGroupID) {
		t.Fatalf("minimal item used group %d", minimalItem.ItemIconGroupId)
	}
	if !strings.Contains(minimalItem.IconJson, "https://minimal.example/favicon.ico") {
		t.Fatalf("minimal item has unexpected icon: %s", minimalItem.IconJson)
	}

	refreshed := performJSONRequest(t, internalRouter, "/refreshToken", "", gin.H{})
	if refreshed.Code != 0 {
		t.Fatalf("refresh failed: %+v", refreshed)
	}
	refreshedConfig := configResponse{}
	if err := json.Unmarshal(refreshed.Data, &refreshedConfig); err != nil {
		t.Fatal(err)
	}
	if refreshedConfig.Token == "" || refreshedConfig.Token == enabledConfig.Token {
		t.Fatalf("token was not rotated: %+v", refreshedConfig)
	}
	oldToken := performJSONRequest(t, externalRouter, "/version", enabledConfig.Token, gin.H{})
	if oldToken.Code != 1000 {
		t.Fatalf("old token remained valid: %+v", oldToken)
	}

	disabled := performJSONRequest(t, internalRouter, "/disable", "", gin.H{})
	if disabled.Code != 0 {
		t.Fatalf("disable failed: %+v", disabled)
	}
	disabledToken := performJSONRequest(t, externalRouter, "/version", refreshedConfig.Token, gin.H{})
	if disabledToken.Code != 1000 {
		t.Fatalf("disabled token remained valid: %+v", disabledToken)
	}
}

func TestBrowserExtensionVersion(t *testing.T) {
	if browserExtensionCompatibilityVersion != "1.8.1" {
		t.Fatalf("browser extension compatibility version is %q", browserExtensionCompatibilityVersion)
	}
}
