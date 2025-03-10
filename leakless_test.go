package leakless_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/ysmood/leakless"
	"github.com/ysmood/leakless/lib"
)

var p = filepath.FromSlash

type stamp struct {
	Pid  int
	Time time.Time
}

func TestMain(m *testing.M) {
	binDir := filepath.Join(os.TempDir(), "leakless-"+lib.Version)
	lib.E(os.RemoveAll(binDir))
	lib.E(lib.Mkdir("dist", nil))
	lib.Exec("go", "dist", "build", "../cmd/test")
	lib.Exec("go", "dist", "build", "../cmd/zombie")

	os.Exit(m.Run())
}

func TestBasic(t *testing.T) {
	if !leakless.Support() {
		t.Fail()
	}

	lib.Exec(p("dist/test"), "", "on")

	lib.Sleep(2)
	var s stamp
	var pid int
	_ = lib.ReadJSON(p("tmp/pid"), &s)
	_ = lib.ReadJSON(p("tmp/sub-pid"), &pid)

	if s.Pid == 0 {
		t.Log("zombie pid should not be 0")
		t.Fail()
	}
	if s.Pid != pid {
		t.Log("zombie pid output from itself and guard should be the same")
		t.Fail()
	}
	if time.Since(s.Time) < time.Second {
		t.Log("zombie should be killed")
		t.Fail()
	}
}

func TestErr(t *testing.T) {
	l := leakless.New()
	lib.E(l.Command("not-exists").Start())

	pid := <-l.Pid()
	if pid != 0 {
		t.Fail()
	}
	if !regexp.MustCompile(`executable file not found`).MatchString(l.Err()) {
		t.Fail()
	}
}

func TestZombie(t *testing.T) {
	cmd := exec.Command(p("dist/test"), "off")

	lib.E(cmd.Run())

	lib.Sleep(2)
	var s stamp
	_ = lib.ReadJSON(p("tmp/pid"), &s)

	if s.Pid == 0 {
		t.Fail()
	}
	if time.Since(s.Time) > time.Second {
		t.Fail()
	}
}

func TestRace(t *testing.T) {
	const port = 2978
	wg := sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer leakless.LockPort(port)()
			wg.Done()
		}()
	}

	wg.Wait()
}
