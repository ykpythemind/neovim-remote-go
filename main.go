package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
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
	Run(os.Stdout, os.Args...)
}

func Run(out io.Writer, args ...string) {
	var remoteSend string
	var remoteExpr string
	var remoteWait bool
	var debug bool
	var noStart bool
	var help bool
	var servername string
	flagset := flag.NewFlagSet("neovim-remote", flag.ExitOnError)

	flagset.BoolVar(&noStart, "no-start", false, "")
	flagset.BoolVar(&remoteWait, "remote-wait", false, "Block until all buffers opened by this option get deleted or the process exits.")
	flagset.StringVar(&remoteSend, "remote-send", "", "Send key presses")
	flagset.StringVar(&remoteExpr, "remote-expr", "", "Evaluate expression and print result in shell.")
	flagset.StringVar(&servername, "servername", "", "Set the address to be used. This overrides the default \"/tmp/nvimsocket\" and $NVIM_LISTEN_ADDRESS.'")
	flagset.BoolVar(&debug, "debug", false, "debug")
	flagset.BoolVar(&help, "help", false, "show help")
	flagset.Parse(args[1:])
	nonFlagArgs := flagset.Args()

	if help {
		fmt.Println(`
neovim-remote
		`)
		flag.PrintDefaults()
		os.Exit(0)
	}

	option := Option{
		noStart:    noStart,
		remoteWait: remoteWait,
		remoteSend: remoteSend,
		remoteExpr: remoteExpr,
		servername: os.Getenv("NVIM_LISTEN_ADDRESS"),
	}

	if servername != "" {
		// override default
		option.servername = servername
	}

	runner, err := NewRunner(nonFlagArgs, out, option, debug)
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

func NewRunner(files []string, out io.Writer, option Option, debug bool) (*Runner, error) {
	return &Runner{
		out:       out,
		option:    option,
		files:     files,
		waitCount: 0,
		debug:     debug,
	}, nil
}

type Option struct {
	noStart    bool
	servername string
	remoteWait bool
	remoteSend string
	remoteExpr string
}

type Runner struct {
	option    Option
	out       io.Writer
	files     []string
	waitCount int
	debug     bool
	m         sync.Mutex
}

func (r *Runner) Do() error {
	waitCh := make(chan struct{}, 1)

	if r.option.servername == "" {
		return r.startNewNvim()
	}

	nv, err := nvim.Dial(r.option.servername)
	if err != nil {
		var e net.Error
		if errors.As(err, &e) {
			fmt.Println("neterr")
			return r.startNewNvim()
		} else {
			return fmt.Errorf("failed to dial %s: %w", r.option.servername, err)
		}
	}
	defer nv.Close()

	// TODO: nv.SetClientInfo("neovim-remote-go")

	// TODO: fix
	if err := nv.Command("split"); err != nil {
		return err
	}

	for i, file := range r.files {
		editcmd := "edit" // TODO: depends on flag

		if i == 0 {
			// if first, create new buffer with edit (???)
			editcmd = "edit"
		}
		cmd := fmt.Sprintf("%s %s", editcmd, file)

		if err := nv.Command(cmd); err != nil {
			return err
		}

		if r.wait() {
			fmt.Println("dowait")
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
					fmt.Println("received BufDelete")
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
	}

	if r.option.remoteSend != "" {
		_, err := nv.Input(r.option.remoteSend)
		if err != nil {
			return err
		}
	}

	if r.option.remoteExpr != "" {
		// TODO:
		// if options.remote_expr == '-':
		//     options.remote_expr = sys.stdin.read()

		var result interface{}

		err := nv.Eval(r.option.remoteExpr, &result)
		if err != nil {
			return err
		}

		// TODO: another type...
		if s, ok := result.(string); ok {
			fmt.Fprintf(r.out, s)
			return nil
		} else {
			return fmt.Errorf("enexpected Eval result: %+v", result)
		}
	}

	if r.waitCount > 0 {
		fmt.Println("waiting remote buffer delete...")
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

func (r *Runner) doFilenameEscapedCommand(nv *nvim.Nvim, cmd, path string) error {
	// TODO: escape filename for nvim
	// path = self.server.funcs.fnameescape(path)

	// TODO: shortmess
	// shortmess = self.server.options['shortmess']
	// self.server.options['shortmess'] = shortmess.replace('F', '')
	// self.server.command('{} {}'.format(cmd, path))
	// self.server.options['shortmess'] = shortmess

	return nv.Command(fmt.Sprintf("%s %s", cmd, path))
}

func (r *Runner) wait() bool {
	return r.option.remoteWait
}

func (r *Runner) addWait(n int) {
	r.m.Lock()
	r.waitCount += n
	r.m.Unlock()
}

func (r *Runner) startNewNvim() error {
	if r.option.noStart {
		return nil
	}

	fmt.Fprintln(os.Stderr, "Starting new nvim process...")

	env := os.Environ()
	if r.option.servername != "" {
		env = append(env, fmt.Sprintf("NVIM_LISTEN_ADDRESS=%s", r.option.servername))
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
	return syscall.Exec(binary, newArgs, env)
}
