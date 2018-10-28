package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	log.SetPrefix("autobuild: ")
	if _, err := fetch(); err != nil {
		log.Fatalf("fetch: %v", err)
	}
	update := make(chan bool, 1)
	go watch(update)
	for {
		kill, err := run()
		if err != nil {
			log.Printf("run: %v", err)
		}
		<-update
		if err == nil {
			kill()
		}
	}
}

func run() (kill func(), err error) {
	if _, err := cmd("go", "get", "-d"); err != nil {
		return nil, err
	}
	if _, err := cmd("go", "build"); err != nil {
		return nil, err
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("./" + filepath.Base(wd))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return func() { cmd.Process.Signal(os.Interrupt) }, nil
}

func watch(update chan bool) {
	for range time.Tick(time.Minute) {
		updated, err := fetch()
		if err != nil {
			log.Printf("fetch: %v", err)
			continue
		}
		if updated {
			select {
			case update <- true:
			default:
			}
		}
	}
}

func fetch() (updated bool, err error) {
	if _, err := cmd("git", "fetch"); err != nil {
		return false, err
	}
	origin, err := cmd("git", "rev-parse", "origin/master")
	if err != nil {
		return false, err
	}
	master, err := cmd("git", "rev-parse", "master")
	if err != nil {
		return false, err
	}
	if origin == master {
		return false, nil
	}
	b, err := cmd("git", "log", "-n", "1", "origin/master")
	if err != nil {
		return false, err
	}
	b = strings.Replace(strings.TrimSpace(b), "\n", "\n\t", -1)
	log.Printf("origin/master at %s", b)
	_, err = cmd("git", "reset", "--hard", "origin/master")
	return true, err
}

func cmd(arg0 string, args ...string) (string, error) {
	cmd := exec.Command(arg0, args...)
	b, err := cmd.CombinedOutput()
	if err != nil && len(b) > 0 {
		s := arg0
		if len(args) > 0 {
			s += " " + strings.Join(args, " ")
		}
		return "", fmt.Errorf("%s failed:\n%s%v", s, b, err)
	}
	return string(b), err
}
