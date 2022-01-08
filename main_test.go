package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/creack/pty"
)

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func newCommand(cmd ...string) *exec.Cmd {
	c := []string{"-c"}

	c = append(c, cmd...)
	return exec.Command("/bin/sh", c...)
}

func newNvim() *exec.Cmd {
	// return exec.Command("nvim", "-nu", "NORC", "--headless")
	return exec.Command("nvim", "-nu", "NORC")
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

func TestOpenFile(t *testing.T) {
	f, err := os.Create("/tmp/test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r := strconv.Itoa(seededRand.Intn(100000))
	listenAddress := fmt.Sprintf("/tmp/neovim-remote-test_%s", r)
	os.Setenv("NVIM_LISTEN_ADDRESS", listenAddress)

	cmd := newNvim()
	cmd.Env = []string{
		fmt.Sprintf("NVIM_LISTEN_ADDRESS=%s", listenAddress),
		"LANG=en_US.UTF-8",
	}

	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Error(err)
	}
	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	t.Log("cmd started")

	time.Sleep(3 * time.Second)

	t.Logf("try to open file using `neovim-remote`... NVIM_LISTEN_ADDRESS: %s", listenAddress)
	testFile := createTmpFile(t)
	Run("neovim-remote", testFile) // open file with neovim-remote

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
