package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"

	"github.com/neovim/go-client/nvim"
)

func main() {
	var remoteWait string
	var debug bool
	flag.StringVar(&remoteWait, "remote-wait", "", "wait remote")
	flag.BoolVar(&debug, "debug", false, "debug")
	flag.Parse()

	address := os.Getenv("NVIM_LISTEN_ADDRESS")

	// if address == "" {
	// 	log.Fatal("missing env NVIM_LISTEN_ADDRESS")
	// }

	runner, err := NewRunner(address, remoteWait, debug)
	if err != nil {
		log.Fatal(err)
	}

	if err = runner.Do(); err != nil {
		if debug {
			fmt.Println("runner fail")
		}

		log.Fatal(err)
	}
}

func NewRunner(address string, remoteWait string, debug bool) (*Runner, error) {
	if remoteWait == "" {
		return nil, errors.New("remote wait is blank. now it is error")
	}

	return &Runner{
		remoteWait: remoteWait,
		waitCount:  0,
		debug:      debug,
	}, nil
}

type Runner struct {
	address    string
	remoteWait string // now it is file.
	// TODO: files string[]
	waitCount int
	debug     bool
	m         sync.Mutex
}

func (r *Runner) Do() error {
	waitCh := make(chan struct{}, 1)

	if r.address == "" {
		return r.startNewNvim()
	}

	nv, err := nvim.Dial(r.address)
	if err != nil {
		var e net.Error
		if errors.As(err, &e) {
			return r.startNewNvim()
		} else {
			return fmt.Errorf("failed to dial %s: %w", r.address, err)
		}
	}
	defer nv.Close()

	// TODO: nv.SetClientInfo("neovim-remote-go")

	if err := nv.Command("split"); err != nil {
		return err
	}

	// TODO: flag.Args() is file list
	file := r.remoteWait
	cmd := fmt.Sprintf("edit %s", file)
	if err := nv.Command(cmd); err != nil {
		return err
	}

	if r.wait() {
		// set wait for current buffer
		_, err := nv.CurrentBuffer()
		if err != nil {
			return err
		}

		c := strconv.Itoa(nv.ChannelID())

		// FIXME: need to Batch execute???

		r.addWait(+1)

		if err := nv.RegisterHandler("BufDelete", func(args ...interface{}) {
			if r.debug {
				fmt.Println("received bufdelete!!!!!")
			}
			waitCh <- struct{}{}
		}); err != nil {
			return err
		}

		if err := nv.Command("augroup nvr"); err != nil {
			return err
		}

		if err := nv.Command(fmt.Sprintf(`autocmd BufDelete <buffer> silent! call rpcnotify(%s, "BufDelete")`, c)); err != nil {
			return err
		}

		// TODO? もとの実装にはある self.server.command('autocmd VimLeave * if exists("v:exiting") && v:exiting > 0 | silent! call rpcnotify({}, "Exit", v:exiting) | endif'.format(chanid))

		if err := nv.Command("augroup END"); err != nil {
			return err
		}
	}

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

func (r *Runner) wait() bool {
	return r.remoteWait != ""
}

func (r *Runner) addWait(n int) {
	r.m.Lock()
	r.waitCount += n
	r.m.Unlock()
}

func (r *Runner) startNewNvim() error {
	fmt.Fprintln(os.Stderr, "Starting new nvim process...")

	env := os.Environ()
	if r.address != "" {
		env = append(env, fmt.Sprintf("NVIM_LISTEN_ADDRESS=%s", r.address))
	}

	binary := os.Getenv("NVIM_CMD")
	if binary == "" {
		path, err := exec.LookPath("nvim")
		if err != nil {
			return errors.New("Could not find executable path")
		}
		binary = path
	}

	args := flag.Args()
	newArgs := make([]string, len(args)+1)
	copy(newArgs[1:], args)
	newArgs[0] = binary
	fmt.Println("newArgs", newArgs)
	return syscall.Exec(binary, newArgs, env)
}
