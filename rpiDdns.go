package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"sort"
)

var config = initConfig()
var loger = initLog()

const (
	logTypeInfo = "[Info] "
	logTypeWarn = "[Warn] "
	LogTypeErr  = "[Err] "
	LogTypeSuc  = "[Suc] "
)

const (
	BCTypeSuc = "#00FF00"
	BCTypeErr = "#DC143C"
)

type Config struct {
	AccessKeyID string `json:"AccessKeyId"`
	AccessKeySecret string `json:"AccessKeySecret"`
	BearyChatAPI string `json:"BearyChatApi"`
	RR string `json:"RR"`
	DomainName string `json:"DomainName"`
	DNSAPI string `json:"DnsApi"`
	DescribeAction string `json:"DescribeAction"`
	UpdateAction string `json:"UpdateAction"`
	PublicIP string `json:"PublicIp"`
	LoopTime int `json:"LoopTime"`
	LogFileName string `json:"LogFileName"`
}

type DomainRecords struct {
	DRecords map[string][]Record `json:"DomainRecords"`
}
type Record struct {
	RR       string `json:"RR"`
	Value    string `json:"Value"`
	RecordId string `json:"RecordId"`
	Type     string `json:"Type"`
}

// 配置文件
func initConfig() *Config {
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalln(err.Error())
		return nil
	}

	c := &Config{}
	err = json.Unmarshal(data, c)
	if err != nil {
		log.Fatalln(err.Error())
		return nil
	}
	return c
}

// 获取公网ip
func getPulicIP() (string, error) {
	res, err := http.Get(config.PublicIP)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	result, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	reg := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	return reg.FindString(string(result)), nil
}

// 日志输出
func initLog() *log.Logger {
	file, err := os.OpenFile(config.LogFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalln("fail to create log file!")
	}
	logger := log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)
	return logger
}

func logPrintln(logType string, v ...interface{}) {
	loger.SetPrefix(logType)
	loger.Println(v)
}

func bearyChatPost(title, url, atext, color string) {
	if color == BCTypeSuc{
		logPrintln(LogTypeSuc, title, atext, url)
	}else{
		logPrintln(LogTypeErr, title, atext, url)
	}
	//
	req := make(map[string]interface{})
	req["text"] = "rpi info"
	req["attachments"] = []interface{}{map[string]string{"title": title, "url": url, "text": atext + "\n" + url, "color": color}}
	bytesData, _ := json.Marshal(req)
	req_new := bytes.NewReader(bytesData)
	request, _ := http.NewRequest("POST", config.BearyChatAPI, req_new)
	request.Header.Set("Content-type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(request)
	if err == nil{
		defer resp.Body.Close()
	}
}

//生成请求body
func createRequestBody(otherMap map[string]string) map[string]string {

	//公共参数
	curTime := time.Now()
	var bodyMap = map[string]string{
		"Format":           "JSON",
		"Version":          "2015-01-09",
		"AccessKeyId":      config.AccessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   strconv.FormatInt(curTime.UTC().Unix(), 10),
		"SignatureVersion": "1.0",
		"Timestamp":        url.QueryEscape(curTime.UTC().Format("2006-01-02T15:04:05Z")),
	}
	//添加请求参数
	for key, value := range otherMap {
		bodyMap[key] = value
	}
	//签名
	bodyMap["Signature"] = url.QueryEscape(signBody(bodyMap))

	return bodyMap
}

//签名
func signBody(body map[string]string) string {
	var keysList []string
	for key := range body {
		keysList = append(keysList, key)
	}
	sort.Strings(keysList)
	//拼接
	var str string = ""
	for i, v := range keysList {
		if i != 0 {
			str += "&"
		}

		value := body[v]
		str += v + "=" + value
	}
	//urlencode
	encodeStr := "GET&" + url.QueryEscape("/") + "&" + url.QueryEscape(str)
	//hmac
	key := []byte(config.AccessKeySecret + "&")
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(encodeStr))
	hmacStr := mac.Sum(nil)
	//base64
	b64Str := base64.StdEncoding.EncodeToString(hmacStr)
	//fmt.Printf(b64Str)
	return b64Str
}

func getUrl(bodyMap map[string]string) ([]byte, error) {
	bm := createRequestBody(bodyMap)
	urlStr := ""
	for k, v := range bm {
		urlStr += "&" + k + "=" + v
	}
	urlStr = config.DNSAPI + "?" + urlStr[1:]
	fmt.Println(urlStr)
	//
	resp, err := http.Get(urlStr)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func getRpiRecordId() (DomainRecords, error) {
	resp, err := getUrl(map[string]string{
		"Action":    config.DescribeAction,
		"DomainName": config.DomainName,
	})
	var response DomainRecords
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	fmt.Println(response)
	return response, nil
}

func setRpiIp(r Record, wwwIp string) error {
	_, err := getUrl(map[string]string{
		"Action":   config.UpdateAction,
		"RecordId": r.RecordId,
		"RR":       r.RR,
		"Type":     r.Type,
		"Value":    wwwIp,
	})
	return err
}

func main() {

	bearyChatPost("start", "", "", BCTypeSuc)
	
	errCount := 0
	//
	for {
		wwwIp, err := getPulicIP()
		if err != nil {
			bearyChatPost("get publicIp error", "", err.Error(), BCTypeErr)
			errCount++
			time.Sleep(time.Duration(1) * time.Minute)
			if errCount == 6 {
				errCount = 0
				time.Sleep(time.Duration(config.LoopTime) * time.Minute)
			}
			continue
		}
		// logPrintln(logTypeInfo, "get publicIp", wwwIp)
		//
		dRecords, err := getRpiRecordId()
		if err != nil {
			bearyChatPost("get record id error", wwwIp, err.Error(), BCTypeErr)
			errCount++
			time.Sleep(time.Duration(1) * time.Minute)
			if errCount == 6 {
				errCount = 0
				time.Sleep(time.Duration(config.LoopTime) * time.Minute)
			}
			continue
		}
		for _, v := range dRecords.DRecords["Record"] {
			if v.RR == config.RR && v.Value != wwwIp {
				err = setRpiIp(v, wwwIp)
				if err != nil {
					bearyChatPost("set ip error", wwwIp, err.Error(), BCTypeErr)
					break
				} else {
					bearyChatPost("set ip success", wwwIp, "", LogTypeSuc)
				}
			}
		}
		errCount = 0
		time.Sleep(time.Duration(config.LoopTime) * time.Minute)
	}

}

