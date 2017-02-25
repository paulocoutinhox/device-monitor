package main

import (
	"bytes"
	"fmt"
	"github.com/pborman/uuid"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"sync"
	"errors"
	"encoding/json"
	"github.com/yosemite-open/go-adb"
	"time"
	"encoding/base64"
	"github.com/prsolucoes/go-minicap-proxy/utils"
	"strconv"
)

type Client struct {
	Id         string
	Socket     *websocket.Conn
	WriteMutex sync.Mutex
}

type Message struct {
	MT   string `json:"mt"`
	Data map[string]interface{} `json:"data"`
}

type Device struct {
	Id             string `json:"id"`
	Serial         string `json:"-"`
	Name           string `json:"name"`
	Screenshot     []byte `json:"-"`
	ScreenshotChan <-chan []byte `json:"-"`
}

var (
	ClientsMutex     sync.Mutex
	Clients          = make([]*Client, 0)
	HttpPort         = "3030"
	AdbClient        *adb.Adb
	AdbDeviceWatcher *adb.DeviceWatcher
	ExitChan         = make(chan bool, 1)
	DevicesMutex     sync.Mutex
	Devices          = make([]*Device, 0)
)

func (This *Message) GetDataValueAsFloat(dataKey string) float64 {
	value, ok := This.Data[dataKey]

	if ok {
		if value != nil {
			finalValue, ok := value.(float64)

			if ok {
				return finalValue
			}
		}
	}

	return 0
}

func (This *Message) GetDataValueAsString(dataKey string) string {
	value, ok := This.Data[dataKey]

	if ok {
		if value != nil {
			switch value := value.(type) {
			case int64:
				return strconv.Itoa(int(value))
			case float64:
				return strconv.FormatFloat(float64(value), 'f', -1, 64)
			case string:
				return string(value)
			case bool:
				return strconv.FormatBool(bool(value))
			default:
				return ""
			}
		}
	}

	return ""
}

func (This *Message) GetDataValueAsInt(dataKey string) int64 {
	value, ok := This.Data[dataKey]

	if ok {
		if value != nil {
			finalValue, ok := value.(int64)

			if ok {
				return finalValue
			}
		}
	}

	return 0
}

func debug(message string) {
	log.Println(fmt.Sprintf("> %s", message))
}

func debugf(format string, params ...interface{}) {
	log.Println(fmt.Sprintf("> "+format, params...))
}

func wsHandler(ws *websocket.Conn) {
	// upgrade connection to websocket
	debug(fmt.Sprintf("New connection from: %+v", ws.RemoteAddr()))

	clientId := uuid.New()
	client := Client{}
	client.Id = clientId
	client.Socket = ws

	addClient(&client)

	debug(fmt.Sprintf("Client Id: %v", clientId))

	// read message sent by client
	for {
		messageRaw := make([]byte, 512)
		messageLength, err := ws.Read(messageRaw)
		messageRaw = bytes.Trim(messageRaw, "\x00")

		if err != nil {
			debugf("Read error: %v", clientId)
			break
		}

		if messageLength > 0 {
			var message Message
			err := json.Unmarshal(messageRaw, &message)

			if err != nil {
				debugf("Error while parse message: %v", err)
				continue
			}

			if message.MT == "tap" {
				deviceId := message.GetDataValueAsString("device")
				device, err := getDeviceById(deviceId)

				if err != nil {
					//debugf("Error while search for device: (%v) %v", deviceId, err)
					continue
				}

				posX := message.GetDataValueAsString("x")
				posY := message.GetDataValueAsString("y")
				
				utils.AdbInputTap(device.Serial, posX, posY)
			} else if message.MT == "device-list" {
				response := &Message{
					MT: "device-list",
					Data: map[string]interface{}{
						"list": Devices,
					},
				}

				sendToClientAsJSON(&client, response)
			} else if message.MT == "screenshot" {
				deviceId := message.GetDataValueAsString("device")
				device, err := getDeviceById(deviceId)

				if err != nil {
					//debugf("Error while search for device: (%v) %v", deviceId, err)
					continue
				}

				imageEncoded := base64.StdEncoding.EncodeToString(device.Screenshot)

				response := &Message{
					MT: "screenshot",
					Data: map[string]interface{}{
						"device": deviceId,
						"image":  imageEncoded,
					},
				}

				sendToClientAsJSON(&client, response)
			} else if message.MT == "swipe" {
				deviceId := message.GetDataValueAsString("device")
				device, err := getDeviceById(deviceId)

				if err != nil {
					//debugf("Error while search for device: (%v) %v", deviceId, err)
					continue
				}

				posX := message.GetDataValueAsString("x")
				posY := message.GetDataValueAsString("y")
				destX := message.GetDataValueAsString("dx")
				destY := message.GetDataValueAsString("dy")
				duration := message.GetDataValueAsString("duration")

				utils.AdbInputSwipe(device.Serial, posX, posY, destX, destY, duration)
			}
		}
	}

	debugf("Connection closed: %v", clientId)
}

func main() {
	setupAdbClient()
	go setupHttpServer()
	go startAdbDeviceWatcher()
	go startScreenshotWatcher()

	<-ExitChan
}

func addClient(client *Client) {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()

	if client == nil {
		debug("Invalid client (addClient)")
		return
	}

	Clients = append(Clients, client)
}

func removeClient(client *Client) {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()

	if client == nil {
		debug("Invalid client (removeClient)")
		return
	}

	for i, c := range Clients {
		if c.Id == client.Id {
			debugf("Client removed: %v", client.Id)
			Clients = append(Clients[:i], Clients[i+1:]...)
		}
	}
}

func addDevice(device *Device) {
	DevicesMutex.Lock()
	defer DevicesMutex.Unlock()

	if device == nil {
		debug("Invalid device (addDevice)")
		return
	}

	Devices = append(Devices, device)
}

func removeDevice(device *Device) {
	DevicesMutex.Lock()
	defer DevicesMutex.Unlock()

	if device == nil {
		debug("Invalid device (removeDevice)")
		return
	}

	for i, d := range Devices {
		if d.Id == device.Id {
			debugf("Device removed: %v", device.Id)
			Devices = append(Devices[:i], Devices[i+1:]...)
		}
	}
}

func removeDeviceBySerial(serial string) {
	DevicesMutex.Lock()
	defer DevicesMutex.Unlock()

	for i, d := range Devices {
		if d.Serial == serial {
			debugf("Device removed: %v", d.Id)
			Devices = append(Devices[:i], Devices[i+1:]...)
		}
	}
}

func getDeviceBySerial(serial string) (*Device, error) {
	DevicesMutex.Lock()
	defer DevicesMutex.Unlock()

	for _, d := range Devices {
		if d.Serial == serial {
			return d, nil
		}
	}

	return nil, errors.New("Device not found")
}

func getDeviceById(deviceId string) (*Device, error) {
	DevicesMutex.Lock()
	defer DevicesMutex.Unlock()

	for _, d := range Devices {
		if d.Id == deviceId {
			return d, nil
		}
	}

	return nil, errors.New("Device not found")
}

func disconnectClient(client *Client) {
	if client == nil {
		debug("Invalid client (disconnectClient)")
		return
	}

	debugf("Disconnected: %v", client.Id)
	removeClient(client)
}

func sendToClientAsJSON(client *Client, data interface{}) error {
	client.WriteMutex.Lock()
	defer client.WriteMutex.Unlock()

	if client == nil {
		debug("Invalid client (sendToClientAsJSON)")
		return errors.New("Invalid client (sendToClientAsJSON)")
	}

	err := websocket.JSON.Send(client.Socket, data)

	if err != nil {
		return err
	}

	return nil
}

func sendToClient(client *Client, data []byte) error {
	client.WriteMutex.Lock()
	defer client.WriteMutex.Unlock()

	if client == nil {
		debug("Invalid client (sendToClient)")
		return errors.New("Invalid client (sendToClient)")
	}

	err := websocket.Message.Send(client.Socket, data)

	if err != nil {
		return err
	}

	return nil
}

func setupHttpServer() {
	debug("Server was started")
	debugf("Open in your browser: http://localhost:%v", HttpPort)

	http.Handle("/ws", websocket.Handler(wsHandler))
	http.Handle("/", http.FileServer(http.Dir("public")))

	err := http.ListenAndServe(fmt.Sprintf(":%v", HttpPort), nil)

	if err != nil {
		debug("Fatal error: " + err.Error())
		ExitChan <- true
	}
}

func setupAdbClient() {
	adb, err := adb.New()

	if err != nil {
		debugf("Error on connect with ADB")
		ExitChan <- true
	}

	AdbClient = adb
	AdbDeviceWatcher = AdbClient.NewDeviceWatcher()
}

func startAdbDeviceWatcher() {
	debug("Device watcher is working in background")

	for event := range AdbDeviceWatcher.C() {
		deviceSerial := event.Serial

		if event.NewState == adb.StateOnline {
			/*
			descriptor := adb.DeviceWithSerial(deviceSerial)
			device := AdbClient.Device(descriptor)
			deviceInfo, err := device.DeviceInfo()

			if err != nil {
				debugf("Error on get device information: %v", err)
				continue
			}
			*/

			newDevice := &Device{
				Id:     uuid.New(),
				Serial: deviceSerial,
			}

			debugf("New device added: %s", newDevice.Serial)
			addDevice(newDevice)
		} else {
			stateName := "state changed"

			if event.NewState == adb.StateDisconnected {
				stateName = "disconnected"
			} else if event.NewState == adb.StateOffline {
				stateName = "offline"
			}

			debugf("Device %s: [%s] %+v", stateName, deviceSerial, time.Now())
			removeDeviceBySerial(deviceSerial)
		}
	}

	if AdbDeviceWatcher.Err() != nil {
		debugf("Error on device watcher: %v", AdbDeviceWatcher.Err())
	}

	go startAdbDeviceWatcher()
}

func startScreenshotWatcher() {
	debug("Screenshot watcher is working in background")

	for {
		for _, device := range Devices {
			deviceSerial := device.Serial
			screenshot, err := utils.AdbGetScreenshot(deviceSerial)

			if err == nil {
				device.Screenshot = screenshot
			}
		}

		time.Sleep(1 * time.Millisecond)
	}
}
