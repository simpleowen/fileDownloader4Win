package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"

	"gopkg.in/AlecAivazis/survey.v1"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows/registry"
)

type app struct {
	Chinese string
	Zip     string
	Dir     string
	Ini     string
	Exe     string
	Desktop bool
}

var appMap map[string]app

func checkErrAtMainFunc(err error) {
	if err != nil {
		log.Println(err)
		fmt.Scanln()
	}
}

func openINI(path string, key string) error {
	iniPath := filepath.Join(path, appMap[key].Dir+"\\"+appMap[key].Ini)
	cmd := exec.Command("cmd", "/C", "start", iniPath)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func genShortcut(goos, arch, user, path, key, deskPath string) error {

	// win7 : C:\Users\Administrator\Desktop
	// winxp: C:\Documents and Settings\Administrator\桌面

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
	defer ole.CoUninitialize()

	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer oleShellObject.Release()
	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer wshell.Release()

	var dst, src string

	dst = deskPath + "\\" + appMap[key].Chinese + ".lnk"
	src = filepath.Join(path, appMap[key].Dir+"\\"+appMap[key].Exe)

	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", dst)
	if err != nil {
		return err
	}
	idispatch := cs.ToIDispatch()
	oleutil.PutProperty(idispatch, "TargetPath", src)
	oleutil.CallMethod(idispatch, "Save")
	return nil
}

func unzipLocalfile(unzip2Dir string, zipFile *os.File, goos string) (bool, error) {

	rc, err := zip.OpenReader(zipFile.Name())
	if err != nil {
		return false, err
	}
	defer rc.Close()

	for _, f := range rc.Reader.File {
		fmt.Println("file in zip: ", f.Name)
		frc, err := f.Open()
		if err != nil {
			return false, err
		}
		defer frc.Close()

		fpath := filepath.Join(unzip2Dir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(fpath), f.Mode())
			lf, err := os.Create(fpath)
			if err != nil {
				return false, err
			}
			defer lf.Close()

			_, err = io.Copy(lf, frc)
			if err != nil {
				return false, err
			}
		}

	}

	return true, nil
}

func downloadZip(url string, download2dir string, key string) (*os.File, error) {
	resp, err := http.Get(url + appMap[key].Zip)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var zipFile *os.File

	zipFile, err = os.Create(download2dir + appMap[key].Zip)

	if err != nil {
		return nil, err
	}
	// defer file.Close()

	_, err = io.Copy(zipFile, resp.Body)
	if err != nil {
		return nil, err
	}

	return zipFile, nil
}

func showApps(apps []string) (result string, err error) {
	var qs = []*survey.Question{
		{
			Name: "answer",
			Prompt: &survey.Select{
				Message: "选择要安装的程序",
				Options: apps,
			},
		},
	}
	var answer string
	err = survey.Ask(qs, &answer)
	if err != nil {
		return "", err
	}
	r := regexp.MustCompile("([a-z]+)")
	key := r.FindString(answer)

	return key, nil

}

func listApps(url string) (apps []string, err error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	appsWithJSON, err := ioutil.ReadAll(resp.Body)

	// var rb []byte
	// _, err = resp.Body.Read(rb)

	if err != nil {
		return nil, err
	}
	if len(appsWithJSON) == 0 {
		return nil, errors.New("程序列表为空")
	}

	// apps = strings.Split(string(con), ";")
	err = json.Unmarshal(appsWithJSON, &appMap)
	if err != nil {
		return nil, err
	}
	for key, value := range appMap {
		apps = append(apps, appMap[key].Chinese+"("+key+")")
	}

	return apps, nil
}

func areYouReady(url string) (result string, err error) {
	// propmt := promptui.Select{
	// 	Label: "是否从 " + url + " 安装程序？",
	// 	Items: []string{"是", "否"},
	// }
	// index, result, err = propmt.Run()
	// if err != nil {
	// 	return 0, "", err
	// }
	// return index, result, nil
	var qs = []*survey.Question{
		{
			Name: "yesorno",
			Prompt: &survey.Select{
				Message: "是否从 " + url + " 安装程序？",
				Options: []string{"是", "否"},
				Default: "是",
			},
		},
	}
	var yesorno string
	err = survey.Ask(qs, &yesorno)
	if err != nil {
		return "", err
	}
	return yesorno, nil

}

func sysInfo() (oSys string, arch string, osuser string, deskPath string, err error) {
	// operationSystem := runtime.GOOS
	arch = runtime.GOARCH
	osu, err := user.Current()
	if err != nil {
		log.Println("get os current user")
		return "", "", "", "", err
	}
	osuser = osu.Username
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion", registry.QUERY_VALUE)
	if err != nil {
		log.Println("registry open SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion ")
		return "", "", "", "", err
	}
	defer k.Close()

	v, _, err := k.GetStringValue("ProductName")
	if err != nil {
		log.Println("registry GetStringValue ProductName")
		return "", "", "", "", err
	}
	oSys = v

	l, err := registry.OpenKey(registry.CURRENT_USER, "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Shell Folders", registry.QUERY_VALUE)
	if err != nil {
		log.Println("registry open SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Shell Folders ")
		return "", "", "", "", err
	}
	defer l.Close()

	v, _, err = l.GetStringValue("Desktop")
	if err != nil {
		log.Println("registry GetStringValue Desktop")
		return "", "", "", "", err
	}
	deskPath = v
	return oSys, arch, osuser, deskPath, nil

}

func main() {

	operationSystem := runtime.GOOS

	url := "http://172.42.1.221:8888"
	// url := "http://localhost:9090"
	url4ListApps := url + "/applist"
	url4Download := url + "/download?app="

	var download2Dir string
	var unzip2Dir string
	var oSys, arch, osuser, deskPath string
	var err error

	download2Dir = "c:\\his\\"
	unzip2Dir = "c:\\"
	oSys, arch, osuser, deskPath, err = sysInfo()
	checkErrAtMainFunc(err)

	result, err := areYouReady(url)
	checkErrAtMainFunc(err)
	if result == "否" {
		os.Exit(0)
	}

	var apps []string
	// apps = make([]string, 20)
	apps, err = listApps(url4ListApps)
	checkErrAtMainFunc(err)
	if apps == nil {
		os.Exit(0)
	}

	key, err := showApps(apps)
	checkErrAtMainFunc(err)

	zipFile, err := downloadZip(url4Download, download2Dir, key)
	checkErrAtMainFunc(err)
	defer zipFile.Close()
	fmt.Println("Download complete...")

	_, err = unzipLocalfile(unzip2Dir, zipFile, operationSystem)
	checkErrAtMainFunc(err)
	fmt.Println("Unzip complete...")

	// dirName := strings.TrimPrefix(strings.TrimSuffix(zipFile.Name(), ".zip"), "c:\\\\")
	// dirName := appMap[key].Dir
	err = openINI(unzip2Dir, key)
	checkErrAtMainFunc(err)

	err = genShortcut(oSys, arch, osuser, unzip2Dir, key, deskPath)
	checkErrAtMainFunc(err)

}
