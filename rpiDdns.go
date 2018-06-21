package main

import (
	"fmt"
	"strconv"
	"time"
	//"net/url"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"sort"
	//"encoding/json"
	//"net/http"
	//"bytes"
	//"io/ioutil"
	//"strings"
	"net/http"
	//"io/ioutil"
	"encoding/json"
	//"io"
	"bytes"
	"io/ioutil"
	"regexp"
)

//config
const (
	AccessKeyId     = ""
	AccessKeySecret = ""
	Dns_Api         = "https://alidns.aliyuncs.com"
	Ip_Api          = "https://www.taobao.com/help/getip.php"
	BearyChat_Api   = ""
	LoopTime        = 30 //分钟
)

func bearyChatPost(text, title, url, atext, color string) {
	req := make(map[string]interface{})
	req["text"] = text
	req["attachments"] = []interface{}{map[string]string{"title": title, "url": url, "text": atext, "color": color}}
	bytesData, _ := json.Marshal(req)
	req_new := bytes.NewReader(bytesData)
	request, _ := http.NewRequest("POST", BearyChat_Api, req_new)
	request.Header.Set("Content-type", "application/json")
	client := &http.Client{}
	resp, _ := client.Do(request)
	defer resp.Body.Close()
}

//生成请求body
func createRequestBody(otherMap map[string]string) map[string]string {

	//公共参数
	curTime := time.Now()
	var bodyMap = map[string]string{
		"Format":           "JSON",
		"Version":          "2015-01-09",
		"AccessKeyId":      AccessKeyId,
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
	//fmt.Println(encodeStr)
	//hmac
	key := []byte(AccessKeySecret + "&")
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
	urlStr = Dns_Api + "?" + urlStr[1:]
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

type DomainRecords struct {
	DRecords map[string][]Record `json:"DomainRecords"`
}
type Record struct {
	RR       string `json:"RR"`
	Value    string `json:"Value"`
	RecordId string `json:"RecordId"`
	Type     string `json:"Type"`
}

func getRpiRecordId() (DomainRecords, error) {
	resp, err := getUrl(map[string]string{
		"Action":     "DescribeDomainRecords",
		"DomainName": "flyzero.cn",
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
		"Action":   "UpdateDomainRecord",
		"RecordId": r.RecordId,
		"RR":       r.RR,
		"Type":     r.Type,
		"Value":    wwwIp,
	})
	return err
}

func getIp() (string, error) {
	res, err := http.Get(Ip_Api)
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

func main() {

	bearyChatPost("rpi info", "start", "", "", "#00FF00")

	errCount := 0

	for {

		fmt.Println("开始循环")
		wwwIp, err := getIp()
		if err != nil {
			bearyChatPost("rpi info", "get ip error", "", err.Error(), "#DC143C")
			errCount++
			if errCount == 6 {
				errCount = 0
				time.Sleep(LoopTime * time.Minute)
			}
			continue
		}
		fmt.Println(wwwIp)

		dRecords, err := getRpiRecordId()
		if err != nil {
			bearyChatPost("rpi info", "get record id error", wwwIp, err.Error(), "#DC143C")
			errCount++
			if errCount == 6 {
				errCount = 0
				time.Sleep(LoopTime * time.Minute)
			}
			continue
		}
		for _, v := range dRecords.DRecords["Record"] {
			if v.RR == "rpi" && v.Value != wwwIp {
				err = setRpiIp(v, wwwIp)
				if err != nil {
					bearyChatPost("rpi info", "set ip error", wwwIp, err.Error(), "#DC143C")
					break
				} else {
					bearyChatPost("rpi info", "set ip success", wwwIp, "", "#00FF00")
				}
			}
		}
		errCount = 0
		time.Sleep(LoopTime * time.Minute)
	}

}
