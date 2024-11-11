package main

import (
    "fmt"
    "net/http"
    "encoding/json"
    "time"
    "github.com/eclipse/paho.mqtt.golang"
)

// PortMap represents each service and its proxy port
var PortMap = map[string]int{
	"Echo":   7,
	"SSH":    22,
	"Telnet": 23,
	"HTTP":   80,
	"Modbus": 502,
	"MQTT":   1883,
	"CoAP":   5683,
}

func main() {
	// Handle the static HTML
	http.HandleFunc("/", serveHTML)
	// Handle AJAX requests for data generation
	http.HandleFunc("/generate", generateData)

	fmt.Println("Server started at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// serveHTML serves the static HTML file
func serveHTML(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Service Selection</title>
		<script>
			function submitForm() {
				const selectedServices = [];
				document.querySelectorAll('input[name="service"]:checked').forEach(checkbox => {
					selectedServices.push(checkbox.value);
				});

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
			}
		</script>
	</head>
	<body>
		<h2>Select Services to Generate Fake Data</h2>
		<form onsubmit="event.preventDefault(); submitForm();">
			<ul>
				<li><input type="checkbox" name="service" value="Echo"> Echo (Port 7)</li>
				<li><input type="checkbox" name="service" value="SSH"> SSH (Port 22)</li>
				<li><input type="checkbox" name="service" value="Telnet"> Telnet (Port 23)</li>
				<li><input type="checkbox" name="service" value="HTTP"> HTTP (Port 80)</li>
				<li><input type="checkbox" name="service" value="Modbus"> Modbus (Port 502)</li>
				<li><input type="checkbox" name="service" value="MQTT"> MQTT (Port 1883)</li>
				<li><input type="checkbox" name="service" value="CoAP"> CoAP (Port 5683)</li>
			</ul>
			<button type="submit">Generate Data</button>
		</form>
		<pre id="output"></pre>
	</body>
	</html>
	`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// generateData handles the data generation based on selected services
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
		port := PortMap[service]
		var fakeData string

		switch service {
		case "mqtt":
			fakeData = generateMQTTData(port)
		case "http":
			fakeData = generateHTTPData(port)
		case "tcp":
			fakeData = generateTCPData(port)
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

func generateMQTTData(port int) string {
    opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://localhost:%d", port))
    opts.SetClientID("go_mqtt_client")

    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        return fmt.Sprintf("Failed to connect to MQTT broker: %v\n", token.Error())
    }
    defer client.Disconnect(250)

    topic := "test/topic"
    payload := "This is a test message"
    token := client.Publish(topic, 0, false, payload)
    token.Wait()

    return fmt.Sprintf("Sent MQTT data to localhost:%d on topic %s\n", port, topic)
}

func generateHTTPData(port int) string {
	return fmt.Sprintf("Sending HTTP data on port %d...\n", port)
}

func generateTCPData(port int) string {
	return fmt.Sprintf("Sending TCP data on port %d...\n", port)
}
