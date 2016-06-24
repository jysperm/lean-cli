package apps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/aisk/wizard"
	"github.com/bitly/go-simplejson"
	"github.com/leancloud/lean/api"
	"github.com/leancloud/lean/utils"
)

var errAppInfoNotFound = errors.New("app info not found")

// App ...
type App struct {
	AppName string
	AppID   string
}

// AppInfo ...
type AppInfo struct {
	AppID     string
	MasterKey string
	AppKey    string
}

func appDirPath(projectPath string) string {
	return filepath.Join(projectPath, ".avoscloud")
}

func appFilePath(projectPath string) string {
	return filepath.Join(appDirPath(projectPath), "apps.json")
}

// GetApps returns the current project's linked apps
func GetApps(projectPath string) (apps []App, err error) {
	content, err := ioutil.ReadFile(appFilePath(projectPath))
	if os.IsNotExist(err) {
		return apps, nil
	}
	if err != nil {
		return
	}

	json, err := simplejson.NewJson(content)
	if err != nil {
		return
	}

	for name, _ID := range json.MustMap() {
		ID := _ID.(string)
		apps = append(apps, App{AppName: name, AppID: ID})
	}

	return
}

func updateAppInfoToLocal(appInfo AppInfo) error {
	var jsonObj *simplejson.Json
	infoPath := filepath.Join(utils.HomeDir(), ".leancloud", "app_keys")
	content, err := ioutil.ReadFile(infoPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		jsonObj = simplejson.New()
	} else {
		jsonObj, err = simplejson.NewJson([]byte(content))
		if err != nil {
			return err
		}
	}
	fmt.Println(jsonObj)
	jsonObj.Set(appInfo.AppID, map[string]string{
		"appKey":    appInfo.AppKey,
		"masterKey": appInfo.MasterKey,
	})
	body, err := jsonObj.EncodePretty()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(infoPath, body, 0600)
}

func getAppInfoFromServer(appID string) (AppInfo, error) {
	masterKey := new(string)
	wizard.Ask([]wizard.Question{
		{
			Content: "请输入应用的 Master Key:",
			Input: &wizard.Input{
				Hidden: true,
				Result: masterKey,
			},
		},
	})

	client := api.Client{
		AppID:     appID,
		MasterKey: *masterKey,
		Region:    api.RegionCN,
	}

	content, err := client.AppDetail()
	if err != nil {
		return AppInfo{}, err
	}
	fmt.Println(content)

	return AppInfo{
		AppID:     appID,
		AppKey:    content.Get("app_key").MustString(),
		MasterKey: *masterKey,
	}, nil
}

func getAppInfoFromLocal(appID string) (AppInfo, error) {
	infoPath := filepath.Join(utils.HomeDir(), ".leancloud", "app_keys")
	content, err := ioutil.ReadFile(infoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return AppInfo{}, errAppInfoNotFound
		}
		return AppInfo{}, err
	}

	jsonObj, err := simplejson.NewJson(content)
	if err != nil {
		return AppInfo{}, err
	}

	_, err = jsonObj.Get(appID).Map()
	if err != nil {
		return AppInfo{}, errAppInfoNotFound
	}

	return AppInfo{
		AppID:     appID,
		AppKey:    jsonObj.Get(appID).Get("appKey").MustString(),
		MasterKey: jsonObj.Get(appID).Get("masterKey").MustString(),
	}, nil
}

// GetAppInfo returns the app's info (with master key)
// and this function will try to get these info's from local
// file system first, or from LeanCloud API server if not found
func GetAppInfo(appID string) (appInfo AppInfo, err error) {
	appInfo, err = getAppInfoFromLocal(appID)
	if err == errAppInfoNotFound {
		appInfo, err = getAppInfoFromServer(appID)
		if err != nil {
			return
		}
		err = updateAppInfoToLocal(appInfo)
		if err != nil {
			return
		}
	}
	return appInfo, nil
}

// AddApp add new app into project's linked apps
func AddApp(projectPath string, name string, ID string) error {
	apps, err := GetApps(projectPath)
	if err != nil {
		return err
	}
	apps = append(apps, App{AppName: name, AppID: ID})

	err = os.Mkdir(appDirPath(projectPath), 0700)
	if err != nil && !os.IsExist(err) {
		return err
	}

	jsonApps := map[string]string{}
	for _, app := range apps {
		jsonApps[app.AppName] = app.AppID
	}
	data, err := json.Marshal(jsonApps)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(appFilePath(projectPath), data, 0700); err != nil {
		return err
	}

	return nil
}