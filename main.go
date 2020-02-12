package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/loadavg"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/network"
	"github.com/mackerelio/go-osstat/uptime"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

const (
	topic = "metrics"
)

var (
	mac string
)

func init() {
	var macs []string
	var err error

	if macs, err = getMacAddr(); err != nil {
		log.Fatalf("Error reading MAC address: %v", err)
	}
	if len(macs) > 0 {
		mac = macs[0]
	}
	log.Infof("publishing with MAC: %s", mac)
}

func connect(clientID string, uri *url.URL) mqtt.Client {
	opts := createClientOptions(clientID, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatalf("Error connecting to MQTT Broker: %v", err)
	}
	return client
}

func createClientOptions(clientID string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetClientID(clientID)
	return opts
}

// func listen(uri *url.URL, topic string) {
// 	client := connect(fmt.Sprintf("%s_%s", mac, "sub"), uri)
// 	client.Subscribe(fmt.Sprintf("%s/+/#", topic), 0, func(client mqtt.Client, msg mqtt.Message) {
// 		msgParts := strings.Split(string(msg.Payload()), ";")
// 		if len(msgParts) == 2 {
// 			log.WithFields(log.Fields{
// 				"topic":     msg.Topic(),
// 				"timestamp": msgParts[0],
// 				"value":     msgParts[1],
// 			}).Info("Received MQTT message")
// 		} else {
// 			log.Warnf("Incorrect msg formatting: %s", string(msg.Payload()))
// 		}
// 	})
// }

func getMacAddr() ([]string, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var as []string
	for _, ifa := range ifs {
		if ifa.Flags&net.FlagLoopback == 0 && ifa.Flags&net.FlagUp == 1 {
			a := ifa.HardwareAddr.String()
			if a != "" {
				as = append(as, a)
			}
		}
	}
	return as, nil
}

func handleErr(t mqtt.Token) {
	if err := t.Error(); err != nil {
		log.Warnf("Error publishing message: %v", err)
	}
}

func main() {
	mqttURL := os.Getenv("MQTT_URL")
	if mqttURL == "" {
		mqttURL = "mqtt://localhost:1883"
	}

	uri, err := url.Parse(mqttURL)
	if err != nil {
		log.Fatal(err)
	}

	//go listen(uri, topic)

	client := connect(fmt.Sprintf("%s_%s", mac, "pub"), uri)
	pubTopic := fmt.Sprintf("%s/%s", topic, mac)
	timer := time.NewTicker(10 * time.Second)
	for t := range timer.C {
		// Convert the time to a Unix Nano timestamp string
		timeStamp := strconv.FormatInt(t.UnixNano(), 10)

		// Collect some stats
		uptime, err := uptime.Get()
		if err != nil {
			log.Errorf("Error getting uptime: %v", err)
			return
		}

		client.Publish(pubTopic+"/uptime", 0, false, timeStamp+";"+strconv.FormatFloat(uptime.Seconds(), 'f', 2, 64))

		memory, err := memory.Get()
		if err != nil {
			log.Errorf("Error getting memory stats: %v", err)
			return
		}

		handleErr(client.Publish(pubTopic+"/memory/total", 0, false, timeStamp+";"+strconv.FormatUint(memory.Total, 10)))
		handleErr(client.Publish(pubTopic+"/memory/used", 0, false, timeStamp+";"+strconv.FormatUint(memory.Used, 10)))
		handleErr(client.Publish(pubTopic+"/memory/cached", 0, false, timeStamp+";"+strconv.FormatUint(memory.Cached, 10)))
		handleErr(client.Publish(pubTopic+"/memory/free", 0, false, timeStamp+";"+strconv.FormatUint(memory.Free, 10)))

		loadavg, err := loadavg.Get()
		if err != nil {
			log.Errorf("Error getting load average stats: %v", err)
			return
		}

		handleErr(client.Publish(pubTopic+"/load/avg1", 0, false, timeStamp+";"+strconv.FormatFloat(loadavg.Loadavg1, 'f', 2, 64)))
		handleErr(client.Publish(pubTopic+"/load/avg5", 0, false, timeStamp+";"+strconv.FormatFloat(loadavg.Loadavg5, 'f', 2, 64)))
		handleErr(client.Publish(pubTopic+"/load/avg15", 0, false, timeStamp+";"+strconv.FormatFloat(loadavg.Loadavg15, 'f', 2, 64)))

		before, err := cpu.Get()
		if err != nil {
			log.Errorf("Error getting cpu stats: %v", err)
			return
		}
		time.Sleep(time.Duration(1) * time.Second)
		after, err := cpu.Get()
		if err != nil {
			log.Errorf("Error getting cpu stats: %v", err)
			return
		}
		total := float64(after.Total - before.Total)

		handleErr(client.Publish(pubTopic+"/cpu/user", 0, false, timeStamp+";"+strconv.FormatFloat(float64(after.User-before.User)/total*100, 'f', 2, 64)))
		handleErr(client.Publish(pubTopic+"/cpu/system", 0, false, timeStamp+";"+strconv.FormatFloat(float64(after.System-before.System)/total*100, 'f', 2, 64)))
		handleErr(client.Publish(pubTopic+"/cpu/idle", 0, false, timeStamp+";"+strconv.FormatFloat(float64(after.Idle-before.Idle)/total*100, 'f', 2, 64)))

		networkStats, err := network.Get()
		if err != nil {
			log.Errorf("Error getting network stats: %v", err)
			return
		}

		for _, n := range networkStats {
			handleErr(client.Publish(pubTopic+"/network/"+n.Name+"/txbytes", 0, false, timeStamp+";"+strconv.FormatUint(n.TxBytes, 10)))
			handleErr(client.Publish(pubTopic+"/network/"+n.Name+"/rxbytes", 0, false, timeStamp+";"+strconv.FormatUint(n.RxBytes, 10)))
		}
	}
}
