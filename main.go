package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

const (
	TOTAL_DAMSELS = 350
	TOTAL_ZEDS = 2
	ZED_SPREAD_RADIUS = 4
)

var listen = flag.String("listen", "", "start server on given address")
var connect = flag.String("connect", "", "connect to server on giving address")
var logfile = flag.String("log", "lgo.log", "log to that file")
var standalone = flag.Bool("standalone", false, "run server as standalone")
var ruleFile = flag.String("rule-file", "", "file with rules")
var ruleSet = &stringSet{}

func init() {
	flag.Var(ruleSet, "rule", "game rule(s) to use")
}

func readRuleFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var rules []string
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " ")
		pos := strings.Index(line, "#")
		if pos > 0 {
			line = line[:pos]
		}

		if line != "" {
			rules = append(rules, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

var defaultServerAddr string

func main() {
	flag.Parse()
	if defaultServerAddr != "" && *connect == "" {
		*connect = defaultServerAddr
	}

	// set up logging
	f, err := os.Create(*logfile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)

	// panic protection
	defer logPanic()

	// seed random
	rand.Seed(time.Now().Unix())

	var attachTo interface {
		AttachPlayer(Render) int
	}

	var rules = &Ruleset{}

	if *ruleFile != "" {
		fileRules, err := readRuleFile(*ruleFile)
		if err != nil {
			log.Printf("main: failed to parse rule file '%s': '%s'", *ruleFile, err)
		}
		*ruleSet = append(*ruleSet, fileRules...)
	}

	if len(*ruleSet) > 0 {
		for _, r := range *ruleSet {
			err := rules.AddRules(r)
			if err != nil {
				log.Printf("main: invalid rule '%s', skipping", r)
			}
		}
	}


	if *connect != "" {
		// connect to remote game
		log.Printf("main: connecting to remote game at '%s'", *connect)
		remote, err := ConnectRemoteGame(*connect)
		if err != nil {
			log.Fatal(err)
		}

		go remote.Run()
		attachTo = remote
	} else {
		// start local game
		if len(*rules) == 0 {
			log.Fatal("no valid rules specified")
		}

		// create dispatcher
		if len(*rules) == 0 {
			if *listen != "" {
				rules.AddRules("single")
			} else {
				rules.AddRules("classic")
			}
		}

		log.Println("main: starting dispatcher")
		dispatcher := NewDispatcher(rules)
		go dispatcher.Run()
		attachTo = dispatcher

		if *listen != "" {
			log.Printf("main: starting server at '%s'", *listen)
			server, err := CreateServer(dispatcher, *listen)
			if err != nil {
				log.Fatal(err)
			}
			go server.Serve()
		}
	}

	if *listen != "" && *standalone {
		fmt.Println("server started on", *listen)
		for {
			time.Sleep(time.Hour)
		}
	} else {
		// create local render
		log.Println("main: creating local render")
		render := NewLocalRender()
		render.Init()

		log.Println("main: attaching to game")
		// attach render (as player)
		attachTo.AttachPlayer(render)

		log.Println("main: ready to play!")
		// run render
		render.Run()
	}
}
