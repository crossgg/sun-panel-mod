package system

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"sun-panel/api/api_v1/common/apiReturn"
	"sun-panel/lib/cmn"

	"github.com/gin-gonic/gin"
)

type About struct {
}

const latestReleaseAPIURL = "https://api.github.com/repos/crossgg/sun-panel-mod/releases/latest"
const latestReleaseWebURL = "https://github.com/crossgg/sun-panel-mod/releases/latest"

type releaseInfo struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
}

func releaseVersionFromRedirect(location string) (string, bool) {
	redirectURL, err := url.Parse(location)
	if err != nil {
		return "", false
	}
	const tagPath = "/crossgg/sun-panel-mod/releases/tag/"
	if strings.HasPrefix(redirectURL.Path, tagPath) {
		tagName, err := url.PathUnescape(strings.TrimPrefix(redirectURL.Path, tagPath))
		return tagName, err == nil && tagName != ""
	}
	if strings.TrimRight(redirectURL.Path, "/") == "/crossgg/sun-panel-mod/releases" {
		return "", true
	}
	return "", false
}

func getLatestReleaseFromWeb(client *http.Client) (releaseInfo, bool, error) {
	webClient := *client
	webClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	req, err := http.NewRequest(http.MethodGet, latestReleaseWebURL, nil)
	if err != nil {
		return releaseInfo{}, false, err
	}
	req.Header.Set("User-Agent", "sun-panel-mod")
	resp, err := webClient.Do(req)
	if err != nil {
		return releaseInfo{}, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusMultipleChoices || resp.StatusCode >= http.StatusBadRequest {
		return releaseInfo{}, false, fmt.Errorf("返回状态码 %d", resp.StatusCode)
	}
	tagName, recognized := releaseVersionFromRedirect(resp.Header.Get("Location"))
	if !recognized {
		return releaseInfo{}, false, fmt.Errorf("未识别的版本跳转地址")
	}
	return releaseInfo{TagName: tagName}, tagName == "", nil
}

func isNewerVersion(latest, current string) bool {
	latestParts := strings.Split(strings.TrimPrefix(latest, "v"), ".")
	currentParts := strings.Split(strings.TrimPrefix(current, "v"), ".")
	partCount := len(latestParts)
	if len(currentParts) > partCount {
		partCount = len(currentParts)
	}

	for index := 0; index < partCount; index++ {
		latestPart := 0
		currentPart := 0
		if index < len(latestParts) {
			latestPart, _ = strconv.Atoi(strings.SplitN(latestParts[index], "-", 2)[0])
		}
		if index < len(currentParts) {
			currentPart, _ = strconv.Atoi(strings.SplitN(currentParts[index], "-", 2)[0])
		}
		if latestPart != currentPart {
			return latestPart > currentPart
		}
	}
	return false
}

func (a *About) Get(c *gin.Context) {
	version := cmn.GetSysVersionInfo()
	apiReturn.SuccessData(c, gin.H{
		"versionName": strings.TrimSpace(version.Version),
		"versionCode": version.Version_code,
	})
}

// CheckUpdate 检查更新
func (a *About) CheckUpdate(c *gin.Context) {
	// 获取当前版本
	currentVersion := cmn.GetSysVersionInfo()
	currentVersionName := strings.TrimSpace(currentVersion.Version)

	// 请求GitHub API获取最新版本信息
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", latestReleaseAPIURL, nil)
	if err != nil {
		apiReturn.Error(c, "创建请求失败: "+err.Error())
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sun-panel-mod")

	resp, err := client.Do(req)
	if err != nil {
		apiReturn.Error(c, "检查更新失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		apiReturn.SuccessData(c, gin.H{
			"hasNewVersion": false,
			"latestVersion": currentVersionName,
			"updateContent": "",
		})
		return
	}

	info := releaseInfo{}
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			apiReturn.Error(c, "读取响应失败: "+err.Error())
			return
		}
		if err := json.Unmarshal(bodyBytes, &info); err != nil {
			apiReturn.Error(c, "解析版本信息失败: "+err.Error())
			return
		}
	} else if resp.StatusCode == http.StatusForbidden {
		var noRelease bool
		info, noRelease, err = getLatestReleaseFromWeb(client)
		if err != nil {
			apiReturn.Error(c, "检查更新失败: "+err.Error())
			return
		}
		if noRelease {
			apiReturn.SuccessData(c, gin.H{
				"hasNewVersion": false,
				"latestVersion": currentVersionName,
				"updateContent": "",
			})
			return
		}
	} else {
		apiReturn.Error(c, fmt.Sprintf("检查更新失败: 返回状态码 %d", resp.StatusCode))
		return
	}

	// 提取版本号
	latestVersionName := strings.TrimPrefix(info.TagName, "v")

	// 使用API返回的更新日志
	updateLog := info.Body
	if len(updateLog) > 500 {
		updateLog = updateLog[:500] + "..."
	}

	// 比较版本号
	hasNewVersion := isNewerVersion(latestVersionName, currentVersionName)

	apiReturn.SuccessData(c, gin.H{
		"hasNewVersion": hasNewVersion,
		"latestVersion": latestVersionName,
		"updateContent": updateLog,
	})
}
