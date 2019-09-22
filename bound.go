package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
)

const NODEDB = "/opt/bcloud/node.db"
const CERTFILE = "/opt/bcloud/client.crt"
const CAFILE = "/opt/bcloud/ca.crt"
const KEYFILE = "/opt/bcloud/client.key"
const BCODEURL = "https://console.bonuscloud.io/api/bcode/getBcodeForOther/?email="
const BINDURL = "https://console.bonuscloud.io/api/web/devices/bind"

type SendData struct {
	Bcode       string `json:"bcode"`
	Email       string `json:"email"`
	Mac_address string `json:"mac_address"`
}
type ResponseData struct {
	Cert struct {
		Cert string `json:"cert"`
		Key  string `json:"key"`
		Ca   string `json:"ca"`
	} `json:"Cert"`
	Message string `json:"message"`
	Code int `json:"code"`
	Details string `json:"details"`
}
type Get_Bcode struct {
	Code int `json:"code"`
	Ret  struct {
		List []struct {
			Bcode  string `json:"bcode"`
			Region int    `json:"region"`
		} `json:"list"`
		Mainland []struct {
			Bcode  string `json:"bcode"`
			Region int    `json:"region"`
		} `json:"mainland"`
		NonMainland []struct {
			Bcode  string `json:"bcode"`
			Region int    `json:"region"`
		} `json:"non_mainland"`
		Calculate []interface{} `json:"calculate"`
	} `json:"ret"`
	Message string `json:"message"`
	Details string `json:"details"`
}
type Location struct {
	Longitude     float64 `json:"longitude"`
	City          string  `json:"city"`
	Timezone      string  `json:"timezone"`
	Offset        int     `json:"offset"`
	Region        string  `json:"region"`
	Asn           int     `json:"asn"`
	Organization  string  `json:"organization"`
	Country       string  `json:"country"`
	IP            string  `json:"ip"`
	Latitude      float64 `json:"latitude"`
	ContinentCode string  `json:"continent_code"`
	CountryCode   string  `json:"country_code"`
	RegionCode    string  `json:"region_code"`
}


func Init()  {
	email := os.Getenv("email")
	bcode := os.Getenv("bcode")
	if email==""  {
		log.Println("not set email")
		//os.Exit(1)
	}

	mac:=getMacAddrs()
	if len(mac)==0 {
		log.Println("not get mac address")
		//os.Exit(2)
	}
	log.Println(mac)
	if bcode=="" {
		bcode=get_bcode(email)
		if bcode=="" {
			log.Println("not set bcode and get bcode faild")
			//os.Exit(4)
		}
	}
	//log.Println("mac:",mac[0],"\temail:",email,"bcode:",bcode)
	bound_post(bcode,email,mac[0])
}
func get_bcode(email string ) string {
	resp,err:=http.Get(BCODEURL+email)
	if err!=nil {
		log.Println("bonud fail: get bcode requests fail:")
		log.Println(err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode!=200 {
		log.Println("get bcode fail:")
		return ""
	}
	body,err:=ioutil.ReadAll(resp.Body)
	if err!=nil {
		log.Println("read body failed")
		return  ""
	}
	var bcodes_ret Get_Bcode
	json.Unmarshal(body,&bcodes_ret)
	if bcodes_ret.Code!=200 {
		log.Println("Unmarshal body failed,raw body:",string(body))
		return ""
	}
	CountryCode:=get_Location()
	log.Println(CountryCode)
	switch CountryCode {
	case "CN":
		if len(bcodes_ret.Ret.Mainland)==0 {
			return ""
		}
		return bcodes_ret.Ret.Mainland[0].Bcode
	case "":
		return ""
	default:
		if len(bcodes_ret.Ret.NonMainland)==0 {
			return ""
		}
		return bcodes_ret.Ret.NonMainland[0].Bcode

	}
}

func get_Location() string {
	resp,err:=http.Get("https://api.ip.sb/geoip")
	if err!=nil {
		log.Println("get location failed")
		log.Println(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var local Location
	json.Unmarshal(body,&local)
	if local.CountryCode!="" {
		return local.CountryCode
	}else {
		return ""
	}
}

func bound_post(bcode, email, mac string) (error) {
	if isBind() {
		log.Println("node already bind")
		return nil
	}
	data := SendData{bcode, email, mac}
	js, err := json.Marshal(data)
	if err!=nil {
		log.Println(err)
	}
	log.Println(string(js))
	resq, err := http.Post(BINDURL, "application/json;charset=utf-8", bytes.NewBuffer(js))
	if err!=nil {
		log.Println("bonud fail:requests fail:")
		log.Println(err.Error())
		return err
	}
	defer resq.Body.Close()
	body, _ := ioutil.ReadAll(resq.Body)
	if resq.StatusCode!=200 {
		log.Println("bonud fail:")
		log.Println(string(body))
		return errors.New("bound fail")
	}
	resp_data := ResponseData{}
	json.Unmarshal(body, &resp_data)
	if resp_data.Code!=200&&resp_data.Code!=0 {
		log.Println("bonud fail")
		return errors.New("bound fail")
	}
	ca_str, err := base64.StdEncoding.DecodeString(resp_data.Cert.Ca)
	err = ioutil.WriteFile(CAFILE, ca_str, 0644)
	if err != nil {
		 return  err
	}
	key_str, err := base64.StdEncoding.DecodeString(resp_data.Cert.Key)
	err = ioutil.WriteFile(KEYFILE, key_str, 0644)
	if err != nil {
		return  err
	}
	cert_str, err := base64.StdEncoding.DecodeString(resp_data.Cert.Cert)
	err = ioutil.WriteFile(CERTFILE, cert_str, 0644)
	if err != nil {
		return  err
	}
	err = ioutil.WriteFile(NODEDB, js, 0644)
	if err != nil {
		return  err
	}
	log.Println("Bound success")
	return nil
}
func getMacAddrs() (macAddrs []string) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return macAddrs
	}

	for _, netInterface := range netInterfaces {
		macAddr := netInterface.HardwareAddr.String()
		if len(macAddr) == 0 {
			continue
		}
		macAddrs = append(macAddrs, macAddr)
	}
	return macAddrs
}
func isBind() (bool) {
	if ! PathExist(CAFILE) {
		return false
	}
	if ! PathExist(KEYFILE) {
		return false
	}
	if ! PathExist(CERTFILE) {
		return false
	}
	if ! PathExist(NODEDB) {
		return false
	}
	return true
}

