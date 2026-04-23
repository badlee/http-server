package main

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	opts := mqtt.NewClientOptions().AddBroker("tcp://127.0.0.1:1884")
	opts.SetClientID("test-client")
	opts.SetConnectTimeout(2 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Connect error: %v", token.Error())
	}
	defer client.Disconnect(250)

	fmt.Println("Connected successfully to MQTT over TCP multiplexer!")

	token := client.Publish("test/topic", 0, false, "Hello from multiplexed MQTT")
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Publish error: %v", token.Error())
	}
	fmt.Println("Message published successfully!")
}
