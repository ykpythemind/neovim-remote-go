package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func newCommand(cmd ...string) *exec.Cmd {
	c := []string{"-c"}

	c = append(c, cmd...)
	return exec.Command("/bin/sh", c...)
}

func newNvim() *exec.Cmd {
	return exec.Command("nvim", "-nu", "NORC", "--headless")
}

func TestOpenFile(t *testing.T) {
	r := strconv.Itoa(seededRand.Intn(100000))
	listenAddress := fmt.Sprintf("/tmp/neovim-remote-test_%s", r)
	os.Setenv("NVIM_LISTEN_ADDRESS", listenAddress)

	cmd := newNvim()
	cmd.Env = []string{fmt.Sprintf("NVIM_LISTEN_ADDRESS=%s", listenAddress)}

	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	cherr := make(chan error, 1)

	go func() {
		cherr <- cmd.Run()
	}()

	time.Sleep(3 * time.Second)

	// do something
	// nv := newCommand("")
	Run("neovim-remote", "README.md")

	time.Sleep(3 * time.Second)

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Error(err)
	}

	select {
	case err := <-cherr:
		if err != nil {
			fmt.Printf("errch: %s", err)
			// t.Fatal(err)
		}
	}

	fmt.Println("out:", outb.String(), "err:", errb.String())
}
