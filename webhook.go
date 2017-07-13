package main

import (
    "os"
	"fmt"
	"net/http"
	"log"
    "strings"
//  "io/ioutil"
    "github.com/bitly/go-simplejson"
    //"encoding/json"
    "github.com/coreos/go-etcd/etcd"
)

var token string
var etcdURL string
var centerPort string

var hostName string
var domain string
var port string
var apiClient *etcd.Client

func initAPI() {
	apiClient = etcd.NewClient([]string{etcdURL})
}

func config() {
    token = os.Getenv("HUMPBACKWEBHOOK_TOKEN")
    etcdURL = os.Getenv("HUMPBACKWEBHOOK_ETCD")
    centerPort = os.Getenv("HUMPBACKWEBHOOK_CENTER_PORT")
}

func cleanDNS(hostName string) {
	_, err := apiClient.RawDelete(fmt.Sprintf("/skydns/docker/%s", hostName), true, true)
	if err != nil {
		log.Println("clean DNS Error:", err)
	}
}

func cleanGateway(hostName string) {
	_, err := apiClient.RawDelete(fmt.Sprintf("/haproxy-discover/services/%s", hostName), true, true)
	if err != nil {
		log.Println("clean Gateway Error:", err)
	}
}

func bindContainerNameAndIP(hostName string, containerName string, containerIP string) {
	_, err := apiClient.SetDir(fmt.Sprintf("/skydns/docker/%s", hostName), 0)
	if err != nil {
		log.Println("bind IP Error:", err)
	}

	_, err2 := apiClient.Set(fmt.Sprintf("/skydns/docker/%s/%s", hostName, containerName), fmt.Sprintf("{\"host\": \"%s\"}", containerIP), 0)
	if err2 != nil {
		log.Println("bind IP Error:", err2)
	}
}

func registerGatewayDomain(hostName string, domain string) {
	_, err := apiClient.Set(fmt.Sprintf("/haproxy-discover/services/%s/domain", hostName), domain, 0)
	if err != nil {
		log.Println("register Domain Error:", err)
	}
}

func registerGatewayNode(hostName string, containerName string, containerIP string, containerPort string) {
	_, err := apiClient.Set(fmt.Sprintf("/haproxy-discover/services/%s/upstreams/%s", hostName, containerName), fmt.Sprintf("%s:%s", containerIP, containerPort), 0)
	if err != nil {
		log.Println("register Node Error:", err)
	}
}

func getContainersInfoByServerName(serverName string) (* simplejson.Json){
	client := new(http.Client)
        req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s:%s/dockerapi/v2/containers?all=true", serverName, centerPort), nil)
        resp, _ := client.Do(req)	
	defer resp.Body.Close()
        containerJs, _ := simplejson.NewFromReader(resp.Body)

	return containerJs
}

func webhook(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    httpToken := r.Header.Get("X-Humpback-Token")
    if len(token) > 0 {
 	 if len(httpToken) == 0 || token != httpToken {
		log.Println("error:", "invalid token")
		fmt.Fprintf(w, "{\"code\":403, \"msg\":\"invalid token\"}")
		return
    	}
    }

    defer r.Body.Close()

    js, _ := simplejson.NewFromReader(r.Body)

    hostName = js.Get("MetaBase").Get("Config").Get("HostName").MustString()
    log.Println("hostname:", hostName)

    cleanDNS(hostName)
    cleanGateway(hostName)

    env := js.Get("MetaBase").Get("Config").Get("Env").MustArray()
    for _, envItem := range env {
	s := strings.Split(envItem.(string), "=")
        if len(s) > 0 {
		switch {
		    case s[0] == "DOMAIN" :
	           	 domain = s[1]
		    case s[0] == "PORT" :
	           	 port = s[1]
		}
	}
    } 
    log.Println("domain:", domain)
    log.Println("port:", port)
    registerGatewayDomain(hostName, domain)

    containers := js.Get("HookContainers").MustArray()
    if len(containers) == 0 {
		log.Println("error:", "container not found")
		fmt.Fprintf(w, "{\"code\":404, \"msg\":\"container not found\"}")
		return
    }

    var serverNameArr []string
    var containerIdArr []string
    for i, _ := range containers {
	serverName := js.Get("HookContainers").GetIndex(i).Get("IP").MustString()
	serverNameArr = append(serverNameArr, serverName)

	containerId := js.Get("HookContainers").GetIndex(i).Get("Container").Get("Id").MustString()
	containerIdArr = append(containerIdArr, containerId)

	log.Println("serverName", serverName)
	log.Println("containerId", containerId)
    }

    for _, serverName := range serverNameArr {
	containersInfo := getContainersInfoByServerName(serverName)	

	containerArr, _ := containersInfo.Array()

	for i, _ := range containerArr {
		containerInfo := containersInfo.GetIndex(i)
		if "bridge" == containerInfo.Get("HostConfig").Get("NetworkMode").MustString() {
			containerIP := containerInfo.Get("NetworkSettings").Get("Networks").Get("bridge").Get("IPAddress").MustString()
			if len(containerIP) > 0 {
				containerName := formatName(containerInfo.Get("Names").GetIndex(0).MustString())
				containerId := containerInfo.Get("Id").MustString()
				for _, id := range containerIdArr {
					if id == containerId {
						log.Println("containerIP", containerIP)
						bindContainerNameAndIP(hostName, containerName, containerIP)
						registerGatewayNode(hostName, containerName, containerIP, port)
					}
				}
			}
		}
	}
    }


    fmt.Fprintf(w, "{\"code\":200\"}")
}

func formatName(name string) (formattedName string) {
    name = strings.Replace(name, "/", "", -1)
    name = strings.ToLower(name)
    return name
}

func main() {
    config()
    initAPI()
    http.HandleFunc("/", webhook)
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}

