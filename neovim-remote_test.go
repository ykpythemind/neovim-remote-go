package neovim_remote

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func TestOpenFile(t *testing.T) {
	f, err := os.Create("/tmp/test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err, cmd, outb, _, listenAddress := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ptmx.Close() }() // Best effort.
	time.Sleep(3 * time.Second)

	t.Log("cmd started")

	t.Logf("try to open file using `neovim-remote`... NVIM_LISTEN_ADDRESS: %s", listenAddress)

	testFile := createTmpFile(t)

	buf := bytes.NewBuffer([]byte(""))
	Run(buf, "neovim-remote", testFile) // open file with neovim-remote

	time.Sleep(1 * time.Second)

	t.Logf("try to edit file %s which opened by `neovim-remote`", testFile)

	ptmx.Write([]byte("ggifugapiyo"))
	ptmx.Write([]byte{27}) // ESC

	time.Sleep(200 * time.Millisecond)

	ptmx.Write([]byte{58}) // :
	ptmx.Write([]byte{'w'})
	ptmx.Write([]byte{'q'})
	ptmx.Write([]byte{33}) // !
	ptmx.Write([]byte{13}) // ENTER

	time.Sleep(100 * time.Millisecond)

	str := outb.String()
	fmt.Fprint(f, str)

	t.Log("check file...")

	got := readFromTmpFile(t, testFile)

	if got != "fugapiyoTEST\n" {
		t.Errorf("neovim-remote did not work as expected...? got %s", got)
	}
}

/*
func TestOpenFileWithRemote(t *testing.T) {
	f, err := os.Create("/tmp/test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err, cmd, outb, _, listenAddress := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	ptmx, err := pty.Start(cmd)
	// TODO: pty.Open
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ptmx.Close() }() // Best effort.
	time.Sleep(3 * time.Second)

	t.Log("cmd started")

	t.Logf("try to open file using `neovim-remote`... NVIM_LISTEN_ADDRESS: %s", listenAddress)

	testFile := createTmpFile(t)
	Run("neovim-remote", "--remote-wait", testFile) // open file with neovim-remote

	time.Sleep(1 * time.Second)

	t.Log("try to edit file which opened by `neovim-remote`")

	ptmx.Write([]byte("ggifugapiyo"))
	ptmx.Write([]byte{27}) // ESC

	time.Sleep(100 * time.Millisecond)

	ptmx.Write([]byte{58}) // :
	ptmx.Write([]byte{'w'})
	ptmx.Write([]byte{'q'})
	ptmx.Write([]byte{33}) // !
	ptmx.Write([]byte{13}) // ENTER

	time.Sleep(100 * time.Millisecond)

	str := outb.String()
	fmt.Fprint(f, str)

	t.Log("check file...")

	got := readFromTmpFile(t, testFile)

	if got != "fugapiyoTEST\n" {
		t.Errorf("neovim-remote did not work as expected...? got %s", got)
	}
}
*/

func TestRemoteSend(t *testing.T) {
	listenAddress := randomSocket()
	cmd, _, _ := newNvim(true, listenAddress)

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	buf := bytes.NewBuffer([]byte(""))
	Run(buf, "neovim-remote", "--nostart", "--remote-send", "iabc<CR><esc>")
	Run(buf, "neovim-remote", "--nostart", "--remote-expr", "getline(1)")

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Error(err)
	}

	_ = cmd.Wait()

	got := buf.String()
	want := "abc"
	if want != got {
		t.Errorf("result is not expected. want %s, but got %s", want, got)
	}
}

func newNvim(headless bool, listenAddress string) (cmd *exec.Cmd, stdout *bytes.Buffer, stderr *bytes.Buffer) {
	if headless {
		cmd = exec.Command("nvim", "-nu", "NORC", "--headless")
	} else {
		cmd = exec.Command("nvim", "-nu", "NORC")
	}

	cmd.Env = []string{
		fmt.Sprintf("NVIM_LISTEN_ADDRESS=%s", listenAddress),
		"LANG=en_US.UTF-8",
	}

	os.Setenv("NVIM_LISTEN_ADDRESS", listenAddress)

	stdb := make([]byte, 1024)
	stdout = bytes.NewBuffer(stdb)
	stde := make([]byte, 1024)
	stderr = bytes.NewBuffer(stde)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return
}

func randomSocket() string {
	r := strconv.Itoa(seededRand.Intn(100000))
	return fmt.Sprintf("/tmp/neovim-remote-test_%s", r)
}

func setup(t *testing.T) (err error, cmd *exec.Cmd, stdout *bytes.Buffer, stderr *bytes.Buffer, listenAddress string) {
	listenAddress = randomSocket()

	cmd, o, e := newNvim(false, listenAddress)

	stdout = o
	stderr = e
	return
}

func createTmpFile(t *testing.T) (filename string) {
	t.Helper()

	tmpFile, err := ioutil.TempFile("", "tmptest")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()

	tmpFile.WriteString("TEST")

	return tmpFile.Name()
}

func readFromTmpFile(t *testing.T, filename string) string {
	t.Helper()

	f, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}
