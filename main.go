package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

//go:embed config.json
var configBytes []byte

func randomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:53.0) Gecko/20100101 Firefox/53.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.79 Safari/537.36 Edge/14.14393",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.131 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:66.0) Gecko/20100101 Firefox/66.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.157 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 6.2; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/68.0.3440.106 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.131 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:67.0) Gecko/20100101 Firefox/67.0",
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.131 Safari/537.36",
	}

	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(userAgents))

	return userAgents[index]
}

type Product struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	OutOfStockText string `json:"outOfStockText"`
	InStock        bool
}

type Config struct {
	EndHour      int           `json:"endHour"`
	SleepSeconds time.Duration `json:"sleepSeconds"`
	TwilioSID    string        `json:"twilioSID"`
	TwilioToken  string        `json:"twilioToken"`
	SendTo       []string      `json:"sendTo"`
	SendFrom     string        `json:"sendFrom"`
	Products     []*Product    `json:"products"`
}

// Response is a Twilio sms response to be sent as xml
// It can contain any number of text Messages
type Response struct {
	Message []string `xml:Message>Body`
}

// SendText sends an sms message to the specified number
func sendStockNotification(c Config) int {
	// Config for text message
	data := url.Values{}

	var respCode int

	msg := "???? STOCK ALERT ????"
	for _, product := range c.Products {
		if product.InStock {
			msg = fmt.Sprintf("%s\n%s %s\n\n", msg, product.Name, product.URL)
		}
	}

	for _, recipient := range c.SendTo {
		data.Set("To", recipient)
		data.Set("From", c.SendFrom)
		data.Set("Body", msg)

		msgURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", c.TwilioSID)

		// Set up request
		r, _ := http.NewRequest(http.MethodPost, msgURL, strings.NewReader(data.Encode()))
		r.SetBasicAuth(c.TwilioSID, c.TwilioToken)
		r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

		// Send Request
		client := &http.Client{}
		resp, _ := client.Do(r)
		respCode := resp.StatusCode

		if respCode >= 400 {
			break
		}
	}

	return respCode
}

func main() {
	var config Config
	json.Unmarshal(configBytes, &config)

	products := config.Products

	fmt.Println("Loaded config, starting job.")

	for time.Now().Hour() < config.EndHour {
		productsAreInStock := false

		for _, product := range products {
			fmt.Printf("Checking if %v is in stock\n", product.Name)
			req, err := http.NewRequest("GET", product.URL, nil)
			if err != nil {
				log.Fatal(err)
			}

			req.Header.Add("User-Agent", randomUserAgent())
			req.Header.Add("Accept", "text/html, */*; q=0.01")
			req.Header.Add("Accept-Language", "en-CA,en-US;q=0.7,en;q=0.3")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if !strings.Contains(string(body), `<p class="product-out-of-stock">Out of stock</p>`) {
				product.InStock = true
				fmt.Println("Product in Stock!")
				if !productsAreInStock {
					productsAreInStock = true
				}
			}
		}
		fmt.Println("Checking complete")

		if productsAreInStock {
			fmt.Printf("Found %v products, sending...\n", len(config.Products))
			res := sendStockNotification(config)
			if res >= 300 {
				log.Fatalf("SMS completed with error %v. Aborting job.\n", res)
			}
			fmt.Println("Sending complete.")
			break
		} else {
			fmt.Println("Nothing to send")
		}

		fmt.Printf("Batch Sent. Sleeping for %v seconds.\n", config.SleepSeconds)
		time.Sleep(config.SleepSeconds * time.Second)
	}
	fmt.Println("Complete!")
}
