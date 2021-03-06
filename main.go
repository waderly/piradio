package main

import (
	"alarm"
	// go get code.google.com/p/gcfg
	"code.google.com/p/gcfg"
	"log"
	"os"
	// "os/exec"
	"flag"
	"player"
	"sayer"
	"syscall"
	"time"
)

/*	the expected config file key-value structure;
	if not matching, main() will panic
	https://code.google.com/p/gcfg/

	Ex. piradio.ini:

	---

	# Path to streams.list file
	[Streams]
	StreamsList = streams.list

	# Path to (JSON) file containing sounds mappings
	[Sounds]
	SoundsFile = sounds.json

	[Volume]
	VolUpStep = 20
	VolDownStep = 20

	[IPC]
	FifoPath = /tmp/gofifo
	...
	---
*/
type Config struct {
	Streams struct {
		StreamsList string
	}
	Sounds struct {
		SoundsFile string
	}
	Volume struct {
		VolUpStep   int
		VolDownStep int
	}
	IPC struct {
		FifoPath string
	}
}

func main() {

	var (
		a         *alarm.Alarm
		s         *sayer.Sayer
		p         *player.Player
		err       error
		input     string
		conf      Config
		confFile  string
		fifo      *os.File
		bytesRead int
		inputKey  []byte
		// keyEventListener *exec.Cmd
		countdownConfigMode bool
		countdownTime       time.Duration
	)

	flag.StringVar(&confFile, "config", "piradio.ini",
		"Configuration file to parse for mandatory and default values")
	flag.Parse()

	// read in config file into struct
	err = gcfg.ReadFileInto(&conf, confFile)
	// if config not as expected, bail
	if err != nil {
		// TODO user feedback first...,
		// then
		panic(err)
	}

	p = player.NewPlayer(conf.Streams.StreamsList)
	s = sayer.NewSayer(conf.Sounds.SoundsFile, p)
	a = alarm.NewAlarm(s, p)
	countdownTime = 0

	// create named pipe (fifo)
	err = syscall.Mkfifo(conf.IPC.FifoPath, syscall.S_IFIFO|0666)
	if err != nil {
		log.Println(conf.IPC.FifoPath, err)
	}

	fifo, err = os.Open(conf.IPC.FifoPath)
	if err != nil {
		log.Printf("Could not acquire control input from %s, aborting (%s).",
			conf.IPC.FifoPath, err)
		os.Exit(1)
	}

	/*
		// gets started externally as of now
		// because we need root permissions
		keyEventListener = exec.Command("sudo", "./key-event", "/dev/input/event0")
		err = keyEventListener.Start()
		if err != nil {
			log.Printf("Could not start key event listener, aborting.")
			os.Exit(1)
		}
	*/

	inputKey = make([]byte, 2)

	for {
		time.Sleep(100 * time.Millisecond)
		bytesRead, err = fifo.Read(inputKey)

		if err == nil && bytesRead == 2 {
			// ignore null bytes
			// (checking this earlier panicked [?])
			if inputKey[0] != 0 {
				input = string(inputKey)
				/* log.Printf("Read from fifo: bytes %v = string (key) <%v>\n", 
				inputKey, input)
				*/

				switch input {
				case "78":
					p.VolumeUp(conf.Volume.VolUpStep)
				case "74":
					p.VolumeDown(conf.Volume.VolDownStep)
				case "79":
					p.NextStreamByNumber(1)
				case "80":
					p.NextStreamByNumber(2)
				case "81":
					p.NextStreamByNumber(3)
				case "75":
					p.NextStreamByNumber(4)
				case "76":
					p.NextStreamByNumber(5)
				case "77":
					p.NextStreamByNumber(6)
				case "71":
					p.NextStreamByNumber(7)
				case "72":
					p.NextStreamByNumber(8)
				case "73":
					p.NextStreamByNumber(9)
				case "14":
					if countdownConfigMode {
						countdownTime = 0
						countdownConfigMode = false
						log.Println("Left countdown config mode")
					} else {
						p.Quit()
						log.Println("Quit")
						os.Exit(0)
					}
				case "96":
					if countdownConfigMode {

						// TODO check if maximum 59m0s exceeded...
						// TODO make tickBegin and tickStep flexible according
						// to given values...
						a.Start(countdownTime, 5*time.Minute, 1*time.Minute)

						// reset
						countdownTime = 0
						countdownConfigMode = false
						log.Println("Started countdown, left countdown config mode")
					} else {
						countdownConfigMode = true
						log.Println("Entered countdown config mode")
					}
				case "82":
					if countdownConfigMode {
						countdownTime += 10 * time.Minute
						log.Printf("Countdown time = %v", countdownTime)
						s.Say(countdownTime.String())
					}
				case "83":
					if countdownConfigMode {
						countdownTime += 1 * time.Minute
						log.Printf("Countdown time = %v", countdownTime)
						s.Say(countdownTime.String())
					}
				}
			}
		} else if err.Error() != "EOF" {
			// "EOF" is expected if no data waiting
			log.Println(err)
		}
	}
}
