package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-ping/ping"
)

const (
	defaultpingUrl = "www.baidu.com"
	defaultRunDir  = "startup"
	defaultTimeout = 60
)

var (
	pingUrl string
	timeout int
	rundir  string
)

func main() {
	flag.StringVar(&pingUrl, "u", defaultpingUrl, "ping url")
	flag.IntVar(&timeout, "t", defaultTimeout, "timeout")
	flag.StringVar(&rundir, "d", "", "run dir")
	flag.Parse()

	if rundir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}

		rundir = filepath.Join(home, defaultRunDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))

	isOK := make(chan struct{}, 1)
	pinger := ping.New(pingUrl)
	if runtime.GOOS == "windows" {
		pinger.SetPrivileged(true)
	}

	pinger.OnRecv = func(pkt *ping.Packet) {
		isOK <- struct{}{}
	}

	pinger.Interval = time.Second / 3

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := pinger.Run(); err != nil {
					log.Println("ping failed, retry")
				}
			}
			time.Sleep(time.Second)
		}
	}()

	select {
	case <-isOK:
		log.Println("network is available")
		pinger.Stop()
		close(isOK)
		cancel()
	case <-ctx.Done():
		log.Println("timeout")
		return
	}

	_, err := os.Stat(rundir)
	if err != nil {
		err := os.Mkdir(rundir, os.ModePerm)
		if err != nil {
			panic(err)
		}

	}

	filepaths := []string{}
	filepath.Walk(rundir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			filepaths = append(filepaths, path)
		}
		return nil
	})

	cmds := []*exec.Cmd{}
	for _, path := range filepaths {
		cmd := exec.Command("cmd.exe", "/C", path)

		if err := cmd.Start(); err != nil {
			log.Println(path + " run failed")
		}
		cmds = append(cmds, cmd)
		log.Println(path + " run success")
	}

	if len(cmds) == 0 {
		log.Println("no run file")
		return
	}

	time.Sleep(time.Second * 1)
	for _, cmd := range cmds {
		cmd.Process.Kill()
	}

	log.Println("all done")
}
