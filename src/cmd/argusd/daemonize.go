// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-20 22:43 (EDT)
// Function: run as a daemon

package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func daemonize() {

	mode := os.Getenv("_argusdmode")
	prog, err := os.Executable()

	if err != nil {
		fmt.Printf("cannot daemonize: %v", err)
		os.Exit(2)
	}

	if mode == "" {
		// initial execution
		// switch to the background
		os.Setenv("_argusdmode", "1")
		p := &os.ProcAttr{}
		os.StartProcess(prog, os.Args, p)
		os.Exit(0)
	}

	syscall.Setsid()

	if mode == "2" {
		// run and be argus
		return
	}

	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	// watch + restart
	for {
		os.Setenv("_argusdmode", "2")
		p, err := os.StartProcess(prog, os.Args, &os.ProcAttr{})
		if err != nil {
			fmt.Printf("cannot start argus: %v", err)
			os.Exit(2)
		}

		stop := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			select {
			case <-stop:
				return
			case n := <-sigchan:
				// pass the signal on through to running argus
				p.Signal(n)
			}
		}()

		st, _ := p.Wait()
		if !st.Exited() {
			continue
		}
		if st.Success() {
			// done
			os.Exit(0)
		}

		close(stop)
		wg.Wait()
		time.Sleep(5 * time.Second)
	}
}
