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
	// return exec.Command("nvim", "-nu", "NORC")
	return exec.Command("nvim", "-nu", "NORC", "--headless")
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

	// cherr := make(chan error, 1)
	// chFile := make(chan *os.File, 1)

	// go func() {
	// 	cherr <- cmd.Run()
	// }()

	ptmx, err := pty.Start(cmd)
	// pty, tty, err := pty.Open(cmd)
	if err != nil {
		t.Error(err)
	}
	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	fmt.Println("wait start cmd...")
	// ptmx := <-chFile
	fmt.Println("ptmx started")

	time.Sleep(3 * time.Second)

	// do something
	// nv := newCommand("")
	Run("neovim-remote", "README.md")

	// buf := make([]byte, 1024)
	ptmx.Write([]byte("ifugafuga"))
	ptmx.Write([]byte{27}) // ESC

	// // 読み込んで表示
	// n, err := ptmx.Read(buf)
	// fmt.Println(string(buf[:n]))
	// if err != nil {
	// 	panic(err)
	// }
	//
	// ptmx.Write([]byte(":q!"))
	//
	time.Sleep(1 * time.Second)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	ptmx.Write([]byte("ggifugafuga"))

	endgame := time.After(5 * time.Second)

	// time.Sleep(1 * time.Second)

	// 読み込んで表示
	buf := make([]byte, 1024)
	_, err = ptmx.Read(buf)
	// fmt.Println(string(buf[:n]))
	if err != nil {
		panic(err)
	}

	// n := 0

	select {
	case <-endgame:
		fmt.Println("time to stop!")

		// fmt.Println("read")
		// nn, err := ptmx.Read(buf)
		// n = nn
		// if err != nil {
		// 	panic(err)
		// }

		// ptmx.Write([]byte{27}) // ESC
		// ptmx.Write([]byte{58}) // :
		// ptmx.Write([]byte{'q'})
		// ptmx.Write([]byte{33}) // !
		// ptmx.Write([]byte{13}) // ENTER

		ptmx.Write([]byte("%print"))
		ptmx.Write([]byte{13}) // ENTER
		ptmx.Write([]byte("q"))
		ptmx.Write([]byte{33}) // !
		ptmx.Write([]byte{13}) // ENTER

		// buf := make([]byte, 1024)
		// n, err := ptmx.Read(buf)
		// if err != nil {
		// 	panic(err)
		// }
		// fmt.Println(string(buf[:n]))

		time.Sleep(1 * time.Second)

		// if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// 	t.Error(err)
		// }

		// if err := ptmx.Close(); err != nil {
		// 	t.Fatal(err)
		// }

		// fmt.Println("out", outb.String())
		// fmt.Println("err", errb.String())

		// case <-ticker.C:
		// 	fmt.Println("out", outb.String())
		// 	fmt.Println("err", errb.String())
	}

	// err = cmd.Wait()
	// if err != nil {
	// 	t.Logf("err type %s", reflect.TypeOf(err))
	// 	if e, ok := err.(*exec.ExitError); ok {
	// 		fmt.Println("code", e.ExitCode())
	// 		fmt.Println("stderr", e.Stderr)
	// 	}
	//
	// 	str := outb.String()
	// 	fmt.Fprint(f, str)
	// 	fmt.Println("out", str)
	// 	fmt.Println("err", errb.String())
	// 	// t.Fatalf("cmd not finish %s", err)
	// }
	fmt.Println("finish...")

	str := outb.String()
	fmt.Fprint(f, str)
	fmt.Println("out", str)
	fmt.Println("err", errb.String())

}
