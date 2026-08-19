package openapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sun-panel/api/api_v1/common/apiReturn"
	"sun-panel/api/api_v1/common/base"
	"sun-panel/global"
	"sun-panel/lib/cmn"
	"sun-panel/models"
	"sun-panel/models/datatype"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm"
)

const maxSavedIconSize = 2 * 1024 * 1024
const browserExtensionCompatibilityVersion = "1.8.1"

type OpenAPI struct{}

type configResponse struct {
	Enabled  bool   `json:"enabled"`
	Token    string `json:"token"`
	Endpoint string `json:"endpoint"`
}

type itemGroupListItem struct {
	OnlyName    string `json:"onlyName"`
	Title       string `json:"title"`
	ItemGroupID uint   `json:"itemGroupID"`
}

type createItemRequest struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	LanURL      string `json:"lanUrl"`
	IconURL     string `json:"iconUrl"`
	Description string `json:"description"`
	ItemGroupID uint   `json:"itemGroupID"`
	IsSaveIcon  bool   `json:"isSaveIcon"`
}

func currentUser(c *gin.Context) (models.User, bool) {
	userInfo, exists := base.GetCurrentUserInfo(c)
	if !exists || userInfo.ID == 0 {
		apiReturn.ErrorNoAccess(c)
		return models.User{}, false
	}
	return userInfo, true
}

func generateToken() (string, error) {
	data := make([]byte, 24)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func endpointForRequest(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]); forwardedProto == "http" || forwardedProto == "https" {
		scheme = forwardedProto
	}

	host := c.Request.Host
	if forwardedHost := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Host"), ",")[0]); forwardedHost != "" {
		host = forwardedHost
	}
	return fmt.Sprintf("%s://%s/openapi/v1", scheme, host)
}

func configData(c *gin.Context, config models.OpenAPIConfig) configResponse {
	return configResponse{
		Enabled:  config.Enabled,
		Token:    config.Token,
		Endpoint: endpointForRequest(c),
	}
}

func saveConfig(c *gin.Context, enabled bool, refreshToken bool) {
	userInfo, ok := currentUser(c)
	if !ok {
		return
	}

	config := models.OpenAPIConfig{}
	err := global.Db.Where("user_id = ?", userInfo.ID).First(&config).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}

	if err == gorm.ErrRecordNotFound {
		config.UserID = userInfo.ID
	}

	if refreshToken || config.Token == "" {
		token, tokenErr := generateToken()
		if tokenErr != nil {
			apiReturn.Error(c, "failed to generate Open API token")
			return
		}
		config.Token = token
	}
	config.Enabled = enabled

	if config.ID == 0 {
		err = global.Db.Create(&config).Error
	} else {
		err = global.Db.Model(&config).Select("token", "enabled").Updates(&config).Error
	}
	if err != nil {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}

	apiReturn.SuccessData(c, configData(c, config))
}

// GetConfig returns the current user's browser-extension access state.
func (a *OpenAPI) GetConfig(c *gin.Context) {
	userInfo, ok := currentUser(c)
	if !ok {
		return
	}

	config := models.OpenAPIConfig{}
	err := global.Db.Where("user_id = ?", userInfo.ID).First(&config).Error
	if err == gorm.ErrRecordNotFound {
		apiReturn.SuccessData(c, configData(c, config))
		return
	}
	if err != nil {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}
	apiReturn.SuccessData(c, configData(c, config))
}

func (a *OpenAPI) Enable(c *gin.Context) {
	saveConfig(c, true, false)
}

func (a *OpenAPI) Disable(c *gin.Context) {
	userInfo, ok := currentUser(c)
	if !ok {
		return
	}

	result := global.Db.Model(&models.OpenAPIConfig{}).
		Where("user_id = ?", userInfo.ID).
		Update("enabled", false)
	if result.Error != nil {
		apiReturn.ErrorDatabase(c, result.Error.Error())
		return
	}

	config := models.OpenAPIConfig{}
	if result.RowsAffected > 0 {
		global.Db.Where("user_id = ?", userInfo.ID).First(&config)
	}
	apiReturn.SuccessData(c, configData(c, config))
}

func (a *OpenAPI) RefreshToken(c *gin.Context) {
	userInfo, ok := currentUser(c)
	if !ok {
		return
	}

	config := models.OpenAPIConfig{}
	err := global.Db.Where("user_id = ?", userInfo.ID).First(&config).Error
	if err == gorm.ErrRecordNotFound || !config.Enabled {
		apiReturn.Error(c, "Open API is not enabled")
		return
	}
	if err != nil {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}
	saveConfig(c, true, true)
}

// Version is used by the extension for connectivity and compatibility checks.
func (a *OpenAPI) Version(c *gin.Context) {
	version := cmn.GetSysVersionInfo()
	apiReturn.SuccessData(c, gin.H{
		"version":      browserExtensionCompatibilityVersion,
		"versionCode":  version.Version_code,
		"panelVersion": strings.TrimSpace(version.Version),
	})
}

func getOrCreateItemGroups(tx *gorm.DB, userID uint) ([]models.ItemIconGroup, error) {
	groups := []models.ItemIconGroup{}
	if err := tx.Order("sort, created_at").Where("user_id = ?", userID).Find(&groups).Error; err != nil {
		return nil, err
	}
	if len(groups) > 0 {
		return groups, nil
	}

	defaultGroup := models.ItemIconGroup{
		Title:  "APP",
		UserId: userID,
		Icon:   "material-symbols:ad-group-outline",
	}
	if err := tx.Create(&defaultGroup).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&models.ItemIcon{}).
		Where("user_id = ?", userID).
		Update("item_icon_group_id", defaultGroup.ID).Error; err != nil {
		return nil, err
	}
	return []models.ItemIconGroup{defaultGroup}, nil
}

func (a *OpenAPI) GetItemGroupList(c *gin.Context) {
	userInfo, ok := currentUser(c)
	if !ok {
		return
	}

	groups := []models.ItemIconGroup{}
	err := global.Db.Transaction(func(tx *gorm.DB) error {
		var err error
		groups, err = getOrCreateItemGroups(tx, userInfo.ID)
		return err
	})
	if err != nil {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}

	list := make([]itemGroupListItem, 0, len(groups))
	for _, group := range groups {
		list = append(list, itemGroupListItem{
			OnlyName:    group.Title,
			Title:       group.Title,
			ItemGroupID: group.ID,
		})
	}
	apiReturn.SuccessListData(c, list, int64(len(list)))
}

func (a *OpenAPI) CreateItem(c *gin.Context) {
	userInfo, ok := currentUser(c)
	if !ok {
		return
	}

	req := createItemRequest{}
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		apiReturn.ErrorParamFomat(c, err.Error())
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.URL = strings.TrimSpace(req.URL)
	req.IconURL = strings.TrimSpace(req.IconURL)
	if req.Title == "" || req.URL == "" {
		apiReturn.ErrorParamFomat(c, "title and url are required")
		return
	}

	itemGroupID := req.ItemGroupID
	if itemGroupID == 0 {
		err := global.Db.Transaction(func(tx *gorm.DB) error {
			groups, err := getOrCreateItemGroups(tx, userInfo.ID)
			if err == nil {
				itemGroupID = groups[0].ID
			}
			return err
		})
		if err != nil {
			apiReturn.ErrorDatabase(c, err.Error())
			return
		}
	} else {
		var groupCount int64
		if err := global.Db.Model(&models.ItemIconGroup{}).
			Where("id = ? AND user_id = ?", itemGroupID, userInfo.ID).
			Count(&groupCount).Error; err != nil {
			apiReturn.ErrorDatabase(c, err.Error())
			return
		}
		if groupCount == 0 {
			apiReturn.ErrorParamFomat(c, "item group does not exist")
			return
		}
	}

	iconSource := req.IconURL
	if iconSource == "" {
		if pageURL, err := url.ParseRequestURI(req.URL); err == nil && pageURL.Host != "" {
			iconSource = fmt.Sprintf("%s://%s/favicon.ico", pageURL.Scheme, pageURL.Host)
		}
	}
	if req.IsSaveIcon && iconSource != "" {
		dataURL, err := fetchImageDataURL(iconSource)
		if err == nil {
			iconSource = dataURL
		}
	}

	icon := datatype.ItemIconIconInfo{ItemType: 2, Src: iconSource}
	iconJSON, err := json.Marshal(icon)
	if err != nil {
		apiReturn.Error(c, err.Error())
		return
	}

	item := models.ItemIcon{
		IconJson:        string(iconJSON),
		Icon:            icon,
		Title:           req.Title,
		Url:             req.URL,
		LanUrl:          strings.TrimSpace(req.LanURL),
		Description:     strings.TrimSpace(req.Description),
		OpenMethod:      2,
		Sort:            9999,
		ItemIconGroupId: int(itemGroupID),
		UserId:          userInfo.ID,
	}
	if err := global.Db.Create(&item).Error; err != nil {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}

	apiReturn.SuccessData(c, item)
}

func fetchImageDataURL(rawURL string) (string, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", fmt.Errorf("invalid image URL")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Get(parsedURL.String())
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("image server returned status %d", response.StatusCode)
	}
	if response.ContentLength > maxSavedIconSize {
		return "", fmt.Errorf("image exceeds 2 MiB")
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maxSavedIconSize+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxSavedIconSize {
		return "", fmt.Errorf("image exceeds 2 MiB")
	}
	contentType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	if !strings.HasPrefix(contentType, "image/") {
		contentType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("URL did not return an image")
	}

	return fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data)), nil
}
