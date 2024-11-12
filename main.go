package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/goburrow/modbus"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/udp"
)

var (
	enableMQTT   = flag.Bool("mqtt", false, "Enable MQTT service")
	enableModbus = flag.Bool("modbus", false, "Enable Modbus service")
	enableCoAP   = flag.Bool("coap", false, "Enable CoAP service")
)

// Config structure holds the configuration for the MQTT and web servers
type Config struct {
	MQTT struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
	} `json:"mqtt"`
	Web struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
	} `json:"web"`
	CoAP struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
	} `json:"coap"`
	ModBus struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
	} `json:"modbus"`
}

// OTData represents the structure of the data we send
type OTData struct {
	Timestamp        time.Time `json:"timestamp"`
	DeviceID         string    `json:"device_id"`
	Temperature      *float64  `json:"temperature,omitempty"`
	Pressure         *float64  `json:"pressure,omitempty"`
	Humidity         *float64  `json:"humidity,omitempty"`
	Vibration        *float64  `json:"vibration,omitempty"`
	PowerConsumption *float64  `json:"power_consumption,omitempty"`
	FlowRate         *float64  `json:"flow_rate,omitempty"`
	Status           string    `json:"status"`
}

var config Config

func main() {
	flag.Parse()

	// Load configuration from config.json
	if err := loadConfig(); err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// Convert flag status into a JSON object for the HTML page to read
	// These values will pre-check the checkboxes in the HTML.
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/generate", generateData)

	// Start the web server
	webAddress := fmt.Sprintf("%s:%d", config.Web.Address, config.Web.Port)
	fmt.Printf("Server started at http://%s\n", webAddress)
	http.ListenAndServe(webAddress, nil)
}

// loadConfig reads the configuration from the config.json file
func loadConfig() error {
	data, err := os.ReadFile("config.json")
	if err != nil {
		return fmt.Errorf("could not read config file: %v", err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("could not parse config file: %v", err)
	}

	return nil
}

// serveHTML serves the static HTML file
func serveHTML(w http.ResponseWriter, r *http.Request) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Service Selection</title>
    <script>
        // Set the initial states based on command-line flags
        const preselectedServices = {
            mqtt: %t,
            modbus: %t,
            coap: %t
        };

        window.onload = function() {
            const savedServices = JSON.parse(localStorage.getItem('selectedServices')) || [];
            document.querySelectorAll('input[name="service"]').forEach(checkbox => {
                if (savedServices.includes(checkbox.value) || preselectedServices[checkbox.value]) {
                    checkbox.checked = true;
                }
            });
            // Start sending data if any service is selected
            if (document.querySelectorAll('input[name="service"]:checked').length > 0) {
                startSendingData();
            }
        };

        function startSendingData() {
            const selectedServices = [];
            document.querySelectorAll('input[name="service"]:checked').forEach(checkbox => {
                selectedServices.push(checkbox.value);
            });

            if (selectedServices.length > 0) {
                intervalId = setInterval(() => {
                    fetch("/generate", {
                        method: "POST",
                        headers: {
                            "Content-Type": "application/json"
                        },
                        body: JSON.stringify({ services: selectedServices })
                    })
                    .then(response => response.text())
                    .then(data => {
                        document.getElementById("output").textContent = data;
                    });
                }, 1000);
            }
        }

        function stopSendingData() {
            clearInterval(intervalId);
        }

        function onCheckboxChange() {
            const selectedServices = [];
            document.querySelectorAll('input[name="service"]:checked').forEach(checkbox => {
                selectedServices.push(checkbox.value);
            });
            localStorage.setItem('selectedServices', JSON.stringify(selectedServices));

            if (selectedServices.length > 0) {
                startSendingData();
            } else {
                stopSendingData();
            }
        }
    </script>
</head>
<body>
    <h2>Select Services to Generate Fake Data</h2>
    <form onsubmit="event.preventDefault();">
        <ul>
            <li><input type="checkbox" name="service" value="mqtt" onchange="onCheckboxChange()"> MQTT</li>
            <li><input type="checkbox" name="service" value="modbus" onchange="onCheckboxChange()"> Modbus</li>
            <li><input type="checkbox" name="service" value="coap" onchange="onCheckboxChange()"> CoAP</li>
        </ul>
    </form>
    <pre id="output"></pre>
</body>
</html>`, *enableMQTT, *enableModbus, *enableCoAP)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// generateData handles the data generation based on selected services
func generateData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse selected services from the request
	var selected struct {
		Services []string `json:"services"`
	}
	if err := json.NewDecoder(r.Body).Decode(&selected); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Simulate sending fake data over selected protocols
	for _, service := range selected.Services {
		switch service {
		case "mqtt":
			generateMQTTData()
		case "modbus":
			generateModbusData()
		case "coap":
			generateCoAPData()
		default:
			fmt.Sprintf("Unknown service: %s\n", service)
		}

		// Simulate a delay for each service
		time.Sleep(500 * time.Millisecond)
	}

	w.Header().Set("Content-Type", "text/plain")
}

// generateMQTTData simulates the data generation and sending to an MQTT broker
func generateMQTTData() string {
	broker := fmt.Sprintf("tcp://%s:%d", config.MQTT.Address, config.MQTT.Port)
	opts := mqtt.NewClientOptions().AddBroker(broker)
	opts.SetClientID("go_mqtt_client")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Sprintf("Failed to connect to MQTT broker on %s:%d: %v\n", config.MQTT.Address, config.MQTT.Port, token.Error())
	}
	defer client.Disconnect(250)

	// Simulate data generation and publishing on each topic
	for i := 1; i <= rand.Intn(10)+1; i++ { // Randomize the number of devices (1-10)
		deviceType := randomDeviceType()                  // Randomly choose a device type
		deviceID := fmt.Sprintf("%s-%02d", deviceType, i) // Generate device IDs like "TempHumidity-01"

		// Create a unique topic based on the device type and device ID
		topic := fmt.Sprintf("ot/device/%s/%s", deviceType, deviceID)

		data := createOTData(deviceID, deviceType)

		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Sprintf("Failed to generate data for device %s: %v\n", deviceID, err)
		}

		token := client.Publish(topic, 0, false, payload)
		token.Wait()
		time.Sleep(500 * time.Millisecond) // Simulate delay between messages
	}

	return "Sending OT MQTT data\n"
}

func generateModbusData() string {
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", config.ModBus.Address, config.ModBus.Port))
	handler.Timeout = 1 * time.Second
	handler.SlaveId = 1
	handler.Logger = log.New(os.Stdout, "modbus: ", log.LstdFlags)

	client := modbus.NewClient(handler)
	err := handler.Connect()
	if err != nil {
		return fmt.Sprintf("Failed to connect to Modbus server on %s:%d: %v\n", config.ModBus.Address, config.ModBus.Port, err)
	}
	defer handler.Close()

	// Simulate data generation and writing to Modbus registers
	for i := 1; i <= rand.Intn(10)+1; i++ { // Randomize the number of devices (1-10)
		deviceType := randomDeviceType()                  // Randomly choose a device type
		deviceID := fmt.Sprintf("%s-%02d", deviceType, i) // Generate device IDs like "TempHumidity-01"

		data := createOTData(deviceID, deviceType)

		// Convert data to Modbus register values
		registers := []uint16{
			uint16(data.Timestamp.Unix() & 0xFFFF),
			uint16((data.Timestamp.Unix() >> 16) & 0xFFFF),
		}

		if data.Temperature != nil {
			registers = append(registers, uint16(*data.Temperature*100))
		}
		if data.Pressure != nil {
			registers = append(registers, uint16(*data.Pressure*100))
		}
		if data.Humidity != nil {
			registers = append(registers, uint16(*data.Humidity*100))
		}
		if data.Vibration != nil {
			registers = append(registers, uint16(*data.Vibration*100))
		}
		if data.PowerConsumption != nil {
			registers = append(registers, uint16(*data.PowerConsumption*100))
		}
		if data.FlowRate != nil {
			registers = append(registers, uint16(*data.FlowRate*100))
		}

		// Write data to Modbus registers
		// Convert []uint16 to []byte
		registerBytes := make([]byte, len(registers)*2)
		for i, reg := range registers {
			registerBytes[i*2] = byte(reg >> 8)
			registerBytes[i*2+1] = byte(reg & 0xFF)
		}
		_, err := client.WriteMultipleRegisters(0, uint16(len(registerBytes)/2), registerBytes)
		if err != nil {
			return fmt.Sprintf("Failed to write data to Modbus for device %s: %v\n", deviceID, err)
		}

		time.Sleep(500 * time.Millisecond) // Simulate delay between messages
	}

	return "Sending OT Modbus data\n"
}

// generateCoAPData simulates CoAP data generation (dummy)
func generateCoAPData() {
	coapAddress := fmt.Sprintf("%s:%d", config.CoAP.Address, config.CoAP.Port)
	sync := make(chan bool)
	co, err := udp.Dial(coapAddress)
	if err != nil {
		log.Fatalf("Error dialing: %v", err)
	}
	num := 0
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	obs, err := co.Observe(ctx, "/some/path", func(req *pool.Message) {
		log.Printf("Got %+v\n", req)
		num++
		if num >= 10 {
			sync <- true
		}
	})
	if err != nil {
		log.Fatalf("Unexpected error '%v'", err)
	}
	<-sync
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	obs.Cancel(ctx)
}

// randomDeviceType randomly selects a device type that sends different data
func randomDeviceType() string {
	deviceTypes := []string{"TempHumidity", "Flow", "Vibration", "Power"}
	return deviceTypes[rand.Intn(len(deviceTypes))]
}

// createOTData generates data for a specific device type
func createOTData(deviceID, deviceType string) OTData {
	status := "Operational"

	data := OTData{
		Timestamp: time.Now(),
		DeviceID:  deviceID,
		Status:    status,
	}

	// Apply device-specific data
	switch deviceType {
	case "TempHumidity":
		// Generate temperature and humidity, triggering alert if in top 15% of range
		temperature := round(rand.Float64()*80+20, 2) // 20 to 100 degrees Celsius
		humidity := round(rand.Float64()*50+30, 1)    // 30 to 80% humidity
		data.Temperature = &temperature
		data.Humidity = &humidity

		// Trigger alert if temperature is in the top 15% of the range
		if temperature > 100*0.90 {
			data.Status = "Alert"
		}

		// Trigger alert if humidity is in the top 15% of the range
		if humidity > 80*0.90 {
			data.Status = "Alert"
		}
	case "Flow":
		// Generate flow rate and trigger alert if in top 15% of range
		flowRate := round(rand.Float64()*250+50, 2) // 50 to 300 L/min
		data.FlowRate = &flowRate
		if flowRate > 300*0.90 {
			data.Status = "Alert"
		}
	case "Vibration":
		// Generate vibration and trigger alert if in top 15% of range
		vibration := round(rand.Float64()*2, 2) // 0 to 2 G (acceleration)
		data.Vibration = &vibration
		if vibration > 2*0.90 {
			data.Status = "Alert"
		}
	case "Power":
		// Generate power consumption and trigger alert if in top 15% of range
		powerConsumption := round(rand.Float64()*100+50, 2) // 50 to 150 W
		data.PowerConsumption = &powerConsumption
		if powerConsumption > 150*0.90 {
			data.Status = "Alert"
		}
	}

	return data
}

// round function rounds a float to the specified number of decimal places
func round(val float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(val*pow) / pow
}
