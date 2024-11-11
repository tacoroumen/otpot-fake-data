package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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
	// Load configuration from config.json
	if err := loadConfig(); err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// Handle the static HTML
	http.HandleFunc("/", serveHTML)
	// Handle AJAX requests for data generation
	http.HandleFunc("/generate", generateData)

	// Start the web server with the loaded configuration
	webAddress := fmt.Sprintf("%s:%d", config.Web.Address, config.Web.Port)
	fmt.Printf("Server started at http://%s\n", webAddress)
	http.ListenAndServe(webAddress, nil)
}

// loadConfig reads the configuration from the config.json file
func loadConfig() error {
	data, err := ioutil.ReadFile("config.json")
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
	html := `<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Service Selection</title>
		<script>
			let intervalId;

			// Start sending data every second
			function startSendingData() {
				const selectedServices = [];
				document.querySelectorAll('input[name="service"]:checked').forEach(checkbox => {
					selectedServices.push(checkbox.value);
				});

				// Make sure there is at least one service selected
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
					}, 1000); // Send every 1000ms (1 second)
				}
			}

			// Stop sending data
			function stopSendingData() {
				clearInterval(intervalId);
			}

			// Monitor checkbox change
			function onCheckboxChange() {
				if (document.querySelector('input[name="service"]:checked')) {
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
				<li><input type="checkbox" name="service" value="mqtt" onchange="onCheckboxChange()"> MQTT (Port 1883)</li>
			</ul>
		</form>
		<pre id="output"></pre>
	</body>
	</html>`
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
	response := "Generated Fake Data:\n"
	for _, service := range selected.Services {
		var fakeData string

		switch service {
		case "mqtt":
			fakeData = generateMQTTData()
		default:
			fakeData = fmt.Sprintf("Unknown service: %s\n", service)
		}

		response += fakeData

		// Simulate a delay for each service
		time.Sleep(500 * time.Millisecond)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(response))
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

	return fmt.Sprint("Sending OT MQTT data\n")
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
