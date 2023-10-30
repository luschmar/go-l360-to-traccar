package main

import (
	"encoding/json"
	"fmt"
	tasks "github.com/madflojo/tasks"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type CircleInfoResponse struct {
	Circles []CircleInfo `json:"circles"`
}

type CircleInfo struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type CircleDetailResponse struct {
	Id      string   `json:"id"`
	Name    string   `json:"name"`
	Members []Member `json:"members"`
}

type Member struct {
	Id        string   `json:"id"`
	FirstName string   `json:"firstName"`
	Location  Location `json:"location"`
	Features  Feature  `json:"features"`
}

type Location struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
	Accuracy  string `json:"accuracy"`
	Timestamp string `json:"timestamp"`
	Battery   string `json:"battery"`
}

type Feature struct {
	Disconnected string `json:"disconnected"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

type AuthRequest struct {
	GrantType string `json:"grant_type"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

var accessToken string = ""
var circleSlice []string = make([]string, 0)
var client http.Client

func main() {
	// transCfg := &http.Transport{
	//	TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
	//}
	//client = http.Client{Transport: transCfg}
	client = http.Client{}

	// Start the Scheduler
	scheduler := tasks.New()
	defer scheduler.Stop()

	// Main Task
	id, err := scheduler.Add(&tasks.Task{
		Interval: time.Duration(5 * time.Minute),
		TaskFunc: func() error {
			// Put your logic here
			if accessToken == "" {
				gainAccessToken()
			}
			getCircleList()
			loopCircles()
			return nil
		},
	})
	if err != nil {
		// Do Stuff
	}

	fmt.Println(id)
	// Wait for a signal to exit the program gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

}

func gainAccessToken() {
	fmt.Printf("Optain access_token\n")
	authResponse := doAuthRequest()

	var auth AuthResponse
	err := json.NewDecoder(authResponse).Decode(&auth)
	if err != nil {
		// Do Stuff
	}
	accessToken = auth.AccessToken
	fmt.Println(auth.AccessToken)
}

func getCircleList() {
	fmt.Printf("GET circles\n")

	circlesResponse := doGet(circleListUrl())

	var circleInfo CircleInfoResponse
	err := json.NewDecoder(circlesResponse).Decode(&circleInfo)
	if err != nil {
		// Do Stuff
		fmt.Println("Error because of decoding! ", err)
	}
	fmt.Println(circleInfo)
	circleSlice = make([]string, len(circleInfo.Circles))
	for j := 0; j < len(circleInfo.Circles); j++ {
		circleSlice[j] = circleInfo.Circles[j].Id
	}
	fmt.Printf("%s\n", circleSlice)
}

func memberToRequest(member Member) string {
	if member.Features.Disconnected == "1" {
		return fmt.Sprintf("http://%s/?id=%s&lat=%s&lon=%s&accuracy=%s&batt=%s&timestamp=%s&valid=false",
			"10.0.0.10:3055",
			member.Id,
			member.Location.Latitude,
			member.Location.Longitude,
			member.Location.Accuracy,
			member.Location.Battery,
			member.Location.Timestamp)
	}

	return fmt.Sprintf("http://%s/?id=%s&lat=%s&lon=%s&accuracy=%s&batt=%s&timestamp=%s",
		"10.0.0.10:3055",
		member.Id,
		member.Location.Latitude,
		member.Location.Longitude,
		member.Location.Accuracy,
		member.Location.Battery,
		member.Location.Timestamp)
}

func authUrl() string {
	return fmt.Sprintf("https://%s/%s", getenv("L360_HOST", "example.com"), "oauth2/token.json")
}

func authAuthorization() string {
	return fmt.Sprintf("Basic %s", getenv("LBASIC", ""))
}

func circleUrl(id string) string {
	return fmt.Sprintf("https://%s/%s/%s", getenv("L360_HOST", "example.com"), "circles", id)
}

func circleListUrl() string {
	return fmt.Sprintf("https://%s/%s", getenv("L360_HOST", "example.com"), "circles")
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func loopCircles() {
	for _, c := range circleSlice {
		circlesResponse := doGet(circleUrl(c))
		var circleDetail CircleDetailResponse
		err := json.NewDecoder(circlesResponse).Decode(&circleDetail)
		if err != nil {
			// Do Stuff
			fmt.Println("Error because of decoding! ", err)
		}
		fmt.Println(circleDetail)

		for j := 0; j < len(circleDetail.Members); j++ {
			member := circleDetail.Members[j]

			if member.Location != (Location{}) {

				url := memberToRequest(member)
				fmt.Println(url)

				resp, err := http.Get(url)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(resp)
			}
		}
	}
}

func doAuthRequest() io.ReadCloser {
	// Create an HTTP request with custom headers
	form := url.Values{
		"grant_type": {"password"},
		"username":   {getenv("LUSER", "")},
		"password":   {getenv("LPASSWORD", "")},
	}

	req, err := http.NewRequest("POST", authUrl(), strings.NewReader(form.Encode()))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return nil
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("user-agent", "com.life360.android.safetymapd")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", authAuthorization())

	// Send the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return nil
	}

	if resp.StatusCode != 200 {
		// TODO: Block execution next hour?
	}

	return resp.Body
}

func doGet(url string) io.ReadCloser {
	req := prepareRequest(url, "GET")

	// Send the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return nil
	}

	if resp.StatusCode == 401 {
		accessToken = ""
		circleSlice = make([]string, 0)
		return nil
	}

	return resp.Body
}

func prepareRequest(url string, requestMethod string) *http.Request {
	// Create an HTTP request with custom headers
	req, err := http.NewRequest(requestMethod, url, nil)
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return nil
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("user-agent", "com.life360.android.safetymapd")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", accessToken))
	return req
}
