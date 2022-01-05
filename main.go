package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/neovim/go-client/nvim"
)

func main() {
	var remoteWait string
	var debug bool
	flag.StringVar(&remoteWait, "remote-wait", "", "wait remote")
	flag.BoolVar(&debug, "debug", false, "debug")
	flag.Parse()

	address := os.Getenv("NVIM_LISTEN_ADDRESS")

	if address == "" {
		log.Fatal("missing env NVIM_LISTEN_ADDRESS")
	}

	runner, err := NewRunner(address, remoteWait, debug)
	if err != nil {
		log.Fatal(err)
	}
	defer runner.Close()

	if err = runner.Do(); err != nil {
		log.Fatal(err)
	}
}

func NewRunner(address string, remoteWait string, debug bool) (*Runner, error) {
	nvim, err := nvim.Dial(address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s %s", address, err)
	}

	// TODO: nvim.SetClientInfo()

	if remoteWait == "" {
		return nil, errors.New("remote wait is blank. now it is error")
	}

	return &Runner{
		nvim:       nvim,
		remoteWait: remoteWait,
		waitCount:  0,
		debug:      debug,
	}, nil
}

type Runner struct {
	nvim       *nvim.Nvim
	remoteWait string // now it is file.
	// TODO: files string[]
	waitCount int
	debug     bool
	m         sync.Mutex
}

func (r *Runner) Do() error {
	waitCh := make(chan struct{}, 1)

	if err := r.nvim.Command("split"); err != nil {
		return err
	}

	// TODO: flag.Args() is file list
	file := r.remoteWait
	cmd := fmt.Sprintf("edit %s", file)
	if err := r.nvim.Command(cmd); err != nil {
		return err
	}

	if r.wait() {
		// set wait for current buffer
		_, err := r.nvim.CurrentBuffer()
		if err != nil {
			return err
		}

		c := strconv.Itoa(r.nvim.ChannelID())

		// need to Batch execute???

		r.addWait(+1)
		// if err := r.nvim.Subscribe("BufDelete"); err != nil {
		// 	return err
		// }

		if err := r.nvim.RegisterHandler("BufDelete", func(args ...interface{}) {
			if r.debug {
				fmt.Println("received bufdelete!!!!!")
			}
			waitCh <- struct{}{}
		}); err != nil {
			return err
		}

		if err := r.nvim.Command("augroup nvr"); err != nil {
			return err
		}

		if err := r.nvim.Command(fmt.Sprintf(`autocmd BufDelete <buffer> silent! call rpcnotify(%s, "BufDelete")`, c)); err != nil {
			return err
		}

		// if err := r.nvim.Command(fmt.Sprintf(`autocmd VimLeave silent! call rpcnotify(%s, "BufDelete")`, c)); err != nil {
		// 	return err
		// }

		// TODO? self.server.command('autocmd VimLeave * if exists("v:exiting") && v:exiting > 0 | silent! call rpcnotify({}, "Exit", v:exiting) | endif'.format(chanid))

		if err := r.nvim.Command("augroup END"); err != nil {
			return err
		}
	}

	// wait logic...

loop:
	for {
		select {
		case <-waitCh:
			r.addWait(-1)
		default:
			// if not wait...

			if r.waitCount < 1 {
				break loop
			}
		}
	}

	return nil
}

func (r *Runner) Close() error {
	return r.nvim.Close()
}

func (r *Runner) wait() bool {
	return r.remoteWait != ""
}

func (r *Runner) addWait(n int) {
	r.m.Lock()
	r.waitCount += n
	r.m.Unlock()
}
