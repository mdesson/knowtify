# knowtify
Run an executable, get notified. Plays nice with cron.

## Installation

**WARNING**: Do NOT share your executable or place it someplace you distrust. This project embeds the config file into the executable, anyone with the binary will be able to access your twilio token, phone numbers, and all data in the config file.

1. Place file named `config.json` in the project's directory.

```
{
	"endHour": INT,
	"sleepSeconds": INT,
	"twilioSID": TWILIO_SID,
	"twilioToken": TWILIO_TOKEN,
	"sendTo": ["+1234567890"],
	"sendFrom": "+1234567890",
	"products": [
		{
			"name": NAME,
			"url": PRODUCT_STOCK_URL,
			"outOfStockText": SEARCH_STRING
		}	
	]
}
```

2. Build or install `main.go` to generate the executabl.

## Usage

1. Create your config file.
2. Set up cron job to run executable.
3. At the desired time knowtify will check if the item is in stock every `sleepSeconds` seconds.
4. If any product is in stock, it will notify you and stop sending.
5. When the time is at least `endHour` the job will terminate.
