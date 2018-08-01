//go:generate esc -o ./www/www.go -pkg www -ignore DS_Store|(.*?)\.md|LICENSE|www\.go|(.*?)\.txt -prefix /www/ ./www

package main

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/adampresley/positivitytracker/www"
	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/flags"
	"github.com/labstack/echo"
	"github.com/peterbourgon/diskv"
	"github.com/sirupsen/logrus"
)

const (
	VERSION string = "0.0.1"

	POSITIVE_KEY string = "positive"
	NEGATIVE_KEY string = "negative"
)

type Config struct {
	Host string `config:"host"`
}

type Options []string

var db *diskv.Diskv
var positiveOptions Options
var negativeOptions Options

func main() {
	var err error

	logger := logrus.New().WithField("who", "PositivityTracker")
	server := echo.New()

	logger.WithField("version", VERSION).Infof("Starting Positivity Tracker server...")

	/*
	 * Setup config
	 */
	config := Config{
		Host: "localhost:9000",
	}

	configLoader := confita.NewLoader(flags.NewBackend())

	if err = configLoader.Load(context.Background(), &config); err != nil {
		logger.WithError(err).Fatalf("Unable to load configuration")
	}

	makePositiveOptions()
	makeNegativeOptions()

	/*
	 * Setup database
	 */
	flatTransform := func(s string) []string { return []string{} }

	db = diskv.New(diskv.Options{
		BasePath:     "data-dir",
		Transform:    flatTransform,
		CacheSizeMax: 1024 * 1024,
	})

	/*
	 * Setup HTTP server
	 */
	server.HideBanner = true
	server.GET("/www/*", echo.WrapHandler(http.FileServer(www.FS(false))))
	server.GET("/", home)
	server.GET("/option/positive", getPositiveOption)
	server.GET("/option/negative", getNegativeOption)
	server.POST("/positive", trackPositive)
	server.POST("/negative", trackNegative)
	server.GET("/positive", getPositive)
	server.GET("/negative", getNegative)

	/*
	 * Once you start me up...
	 */
	go func() {
		if err = server.Start(config.Host); err != nil {
			if err == http.ErrServerClosed {
				logger.Infof("Server shutdown")
			} else {
				logger.WithError(err).Errorf("Error starting HTTP server")
			}
		}
	}()

	/*
	 * Wait for shutdown
	 */
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	<-quit

	logger.Infof("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err = server.Shutdown(ctx); err != nil {
		logger.WithError(err).Errorf("Error shutting down HTTP server")
	}
}

func (o Options) Random() string {
	rand.Seed(time.Now().Unix())
	return o[rand.Intn(len(o))]
}

func makePositiveOptions() {
	positiveOptions = Options{
		"I said something positive!",
		"I'm freaking sunshine and rainbows!",
		"Happy happy joy joy!",
		"Joy to the world!",
		"Lovin' life!",
	}
}

func makeNegativeOptions() {
	negativeOptions = Options{
		"Whah whah",
		"I'm positively negative",
		"Leave me alone!",
		"It's the end of the world as we know it!",
		"Woe is me...",
	}
}

func getPositiveCount() int {
	var err error
	var value []byte

	if value, err = db.Read(POSITIVE_KEY); err != nil {
		writeValue(0, POSITIVE_KEY)
		return 0
	}

	result, _ := strconv.Atoi(string(value))
	return result
}

func getNegativeCount() int {
	var err error
	var value []byte

	if value, err = db.Read(NEGATIVE_KEY); err != nil {
		writeValue(0, NEGATIVE_KEY)
		return 0
	}

	result, _ := strconv.Atoi(string(value))
	return result
}

func incrementPositiveCount() int {
	currentValue := getPositiveCount()
	currentValue++

	writeValue(currentValue, POSITIVE_KEY)
	return currentValue
}

func incrementNegativeCount() int {
	currentValue := getNegativeCount()
	currentValue++

	writeValue(currentValue, NEGATIVE_KEY)
	return currentValue
}

func writeValue(value int, key string) {
	db.Write(key, []byte(strconv.Itoa(value)))
}

func home(ctx echo.Context) error {
	html := `<!DOCTYPE html>
<html lang="en">

<head>
	<meta charset="UTF-8" />
	<meta http-equiv="X-UA-Compatible" content="IE=edge" />
	<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />

	<title>Positivity Tracker</title>

	<link href="/www/fontawesome/css/all.min.css" rel="stylesheet" type="text/css" />
	<link href="/www/positivitytracker/css/styles.css" rel="stylesheet" type="text/css" />
</head>

<body>
	<h1 class="title">How do you feel?</h1>

	<div class="options">
		<div class="option">
			<i class="fas fa-smile positive" id="btnPositive"></i>
			<br />
			<p id="positiveOption"></p>
		</div>

		<div class="option">
			<i class="fas fa-frown negative" id="btnNegative"></i>
			<br />
			<p id="negativeOption"></p>
		</div>
	</div>

	<div class="stats" id="stats"></div>

	<table class="table">
		<thead>
			<th style="width: 50%">Positivity</th>
			<th style="width: 50%">Negativity</th>
		</thead>
		<tbody>
			<td id="positivity"></td>
			<td id="negativity"></td>
		</tbody>
	</table>

	<script src="/www/positivitytracker/js/home.js"></script>
</body>
</html>`

	return ctx.HTML(http.StatusOK, html)
}

func getNegativeOption(ctx echo.Context) error {
	option := negativeOptions.Random()
	return ctx.String(http.StatusOK, option)
}

func getPositiveOption(ctx echo.Context) error {
	option := positiveOptions.Random()
	return ctx.String(http.StatusOK, option)
}

func trackNegative(ctx echo.Context) error {
	newCount := incrementNegativeCount()
	return ctx.String(http.StatusOK, strconv.Itoa(newCount))
}

func trackPositive(ctx echo.Context) error {
	newCount := incrementPositiveCount()
	return ctx.String(http.StatusOK, strconv.Itoa(newCount))
}

func getNegative(ctx echo.Context) error {
	count := getNegativeCount()
	return ctx.String(http.StatusOK, strconv.Itoa(count))
}

func getPositive(ctx echo.Context) error {
	count := getPositiveCount()
	return ctx.String(http.StatusOK, strconv.Itoa(count))
}
