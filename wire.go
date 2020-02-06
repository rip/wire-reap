package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gookit/color"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	auth     = "Bearer your_key_here" // edit or leave blank to login (Bearer key)
	email    = ""                     // edit
	password = ""                     // edit
	verbose  bool
	black    = color.FgBlack.Render
	blue     = color.FgBlue.Render
	cyan     = color.FgCyan.Render
	green    = color.FgGreen.Render
)

type Login struct {
	// success
	Expires_in int
	Access_token,
	User,
	Token_type string
	// fail
	Code int
	Message,
	Label string
}

func login() string {

	// init client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// json
	values := map[string]string{"email": email, "password": password}
	jsonStr, err := json.Marshal(values)
	if err != nil {
		color.Yellow.Println("error, json marshal failed. bad login values:", values)
	}

	// build request
	req, _ := http.NewRequest("POST", "https://prod-nginz-https.wire.com/login", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "")

	// submit request
	resp, err := client.Do(req)
	if err != nil {
		color.Yellow.Println("login request failed")
	}

	// always close everything
	resp.Close = true
	defer resp.Body.Close()

	/* // read resp
	body, _ := ioutil.ReadAll(resp.Body)
	bs := string(body)
	color.Blue.Println(bs)
	auth := bs
	*/

	// Get Auth Bearer Token
	login := new(Login)
	json.NewDecoder(resp.Body).Decode(login)

	switch login.Message {
	case "Logins too frequent":
		color.Red.Println("error: logins too frequent")
		fmt.Printf("%s%s%s\n", green("reuse same"), cyan(" auth "), green("token as much as you can"))
		color.Yellow.Println("try again or use a different account")
		// todo: have a pool of accounts / multi acc support... see https://github.com/rip/go-ghoul
		os.Exit(1)
	case "Authentication failed.":
		color.Red.Println("error: authentication failed, check login and password.")
		os.Exit(1)
	}

	return "Bearer " + login.Access_token

}

func ytc(u string, auth string) {

	// init client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// json
	values := map[string]string{"handle": u}
	jsonStr, err := json.Marshal(values)
	if err != nil {
		color.Yellow.Println("error, json marshal failed. bad line:", u)
	}

	// build request
	req, _ := http.NewRequest("PUT", "https://prod-nginz-https.wire.com/self/handle", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "")
	req.Header.Set("Authorization", auth)

	// submit request
	resp, err := client.Do(req)
	if err != nil {
		color.Yellow.Println(u, "req failed")
		return
	}

	// always close everything
	resp.Close = true
	defer resp.Body.Close()

	switch resp.StatusCode {
	// avail
	case 200:
		color.Green.Println(u)
	// invalid or taken
	case 400, 409:
		if verbose {
			color.Yellow.Println(u)
		}

	default:
		body, _ := ioutil.ReadAll(resp.Body)
		bs := string(body)
		if strings.Contains(bs, "401 Authorization Required") {
			color.Red.Println("auth token fail, leave blank to login for a new one")
			//os.Exit(1) // would have it try to login again but then would have to make sure that threads stop before trying to login or else each thread will login and lock acc.
			auth = login()
		} else {
			color.Red.Println(u, bs)
		}
	}
}

// readLines reads a file into memory and returns a slice of its lines
// appropriated bufio parts from stackoverflow
func readLines(path string) []string {
	// open
	file, err := os.Open(path)
	if err != nil {
		// can use log.Fatal() here but why bother to import the log package for exceptions
		color.Red.Println("error reading " + path + "!")
		color.Yellow.Println("use -h for help")
		os.Exit(1)
	}
	//close
	defer file.Close() // friendly reminder to close files and stuff :)
	// read
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func main() {
	// elite ascii art is a prerequisite for hacking properly
	fmt.Printf("%s%s\n", black("https://github.com/rip/go-reap/"), blue("wire"))
	color.Magenta.Println("  __ _  ___       _ __ ___  __ _ _ __  ")
	color.Magenta.Println(" / _` |/ _ \\ ____| '__/ _ \\/ _` | '_ \\ ")
	color.Magenta.Println("| (_| | (_) |____| | |  __/ (_| | |_) |")
	color.Magenta.Println(" \\__, |\\___/     |_|  \\___|\\__,_| .__/ ")
	color.Magenta.Println(" |___/                          |_|    ")
	// user input
	usernamesPath := flag.String("u", "", "path to file containing usernames")
	threads := flag.Int("t", 9, "amount of simultaneous checking threads")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.Parse()
	// validate input
	if *usernamesPath == "" || *threads < 1 {
		color.Red.Println("-h for help")
		os.Exit(1)
	}
	// authenticatino
	if auth == "" {
		auth = login()
	}
	fmt.Printf("%s%s%s%s\n", green("using "), cyan("auth"), green(": "), cyan(auth))
	// populate slices with requisite data
	usernames := readLines(*usernamesPath)
	// initiate janky golang threadpooling
	semaphore := make(chan bool, *threads)
	// iterate over username file
	for _, username := range usernames {
		semaphore <- true
		// invoke ytc with a goroutine
		go func(username string) {
			// mark thread available after anonymous function has completed
			defer func() { <-semaphore }()
			// ytc this
			ytc(username, auth)
		}(username)
	}
	// clean up thread pool on loop completion
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- true
	}
}
