package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Camera struct {
	xmlURL             string
	name               string
	apiVersion         string
	services           map[string]string
	cameraEndpointURL  string
	availableAPIs      []string
	connected          bool
}

type Device struct {
	XMLName    xml.Name `xml:"urn:schemas-upnp-org:device-1-0 device"`
	FriendlyName   string    `xml:"friendlyName"`
	XDeviceInfo DeviceInfo `xml:"urn:schemas-sony-com:av X_ScalarWebAPI_DeviceInfo"`
}

type DeviceInfo struct {
	XMLName    xml.Name `xml:"urn:schemas-sony-com:av X_ScalarWebAPI_DeviceInfo"`
	APIVersion string   `xml:"X_ScalarWebAPI_Version"`
	ServiceList []Service `xml:"urn:schemas-sony-com:av X_ScalarWebAPI_ServiceList>X_ScalarWebAPI_Service"`
}

type Service struct {
    XMLName xml.Name `xml:"urn:schemas-sony-com:av X_ScalarWebAPI_Service"`
	ServiceType string `xml:"X_ScalarWebAPI_ServiceType"`
	ActionListURL string `xml:"X_ScalarWebAPI_ActionList_URL"`
}

func main() {
	camera := NewCamera()
	fmt.Println(camera)
}

func NewCamera() *Camera {
	camera := &Camera{}
	camera.connected = false

	xmlURL, err := discover()
	if err != nil {
		fmt.Println(err)
	} else {
		camera.xmlURL = xmlURL
		camera.name, camera.apiVersion, camera.services = connect(xmlURL)
		camera.cameraEndpointURL = camera.services["camera"] + "/camera"
		camera.availableAPIs = camera.do("getAvailableApiList")["result"].([]string)

		if stringInSlice("startRecMode", camera.availableAPIs) {
			camera.do("startRecMode")
		}
		camera.availableAPIs = camera.do("getAvailableApiList")["result"].([]string)
		camera.connected = true
	}

	return camera
}

func discover() (string, error) {
	msg := []byte("M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239..255.250:1900\r\n" +
		`MAN: "ssdp:discover"` + " \r\n" +
		"MX: 2\r\n" +
		"ST: urn:schemas-sony-com:service:ScalarWebAPI:1\r\n" +
		"\r\n")

	conn, err := netTimeout("udp", "239.255.255.250:1900", 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, err = conn.Write(msg)
	if != nil {
		return "", err
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1024)

	var receivedData string
	for {
		n, _, err := connFromUDP(buf)
		if err == nil {
			receivedData += string(buf[:n])
			decodedData := strings.Split(receivedData, "\r\n")
			for _, item := range decodedData {
				if strings.Contains(item, "LOCATION") {
					locationURL := strings.Split(strings.TrimSpace(item), " ")[1]
					return locationURL, nil
				}
			}
		} else {
			return "", errors.New("you are not connected to the camera's wifi")
		}
	}
}

func connect(xmlURL string) (string, string,[string]string) {
	deviceXMLRequest, err := http.Get(xmlURL)
	if err != nil {
		fmt.Printf("Failed to get the device XML: %v\n", err)
		return "", "", nil
	}
	defer deviceXMLRequest.Body.Close()

	xmlFile, err := ioutil.ReadAll(deviceXMLRequest.Body)
	if err != nil {
		fmt.Printf("Failed to read the XML: %v\n", err)
		return "", "", nil
	}

	device := Device{}
	err = xml.Unmarshal(xmlFile, &device)
	if err != nil {
		fmt.Printf("Failed to unmarshal XML: %v\n", err)
		return "", "", nil
	}

	name := device.FriendlyName
	apiVersion := device.XDeviceInfo.APIVersion

	apiServiceURLs := make(map[string]string)
	for _, service := range device.XDeviceInfo.ServiceList {
		serviceType := service.ServiceType
		actionURL := service.ActionListURL
		apiServiceURLs[serviceType] = actionURL
	}

	return name, apiVersion, apiServiceURLs
}

func stringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (c *Camera) do(method string, params ...interface{}) map[string]interface{} {
	apiMethod := map[string]interface{}{
		"method":  method,
		"params":  params,
		"id":      1,
		"version": "1.0",
	}

	resp, err := http.PostForm(c.cameraEndpointUrl, url.Values{"data": {apiMethod}})
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	json.Unmarshal(body, &result)
	return result
}
