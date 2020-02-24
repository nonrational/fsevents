// +build darwin

package fsevents

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var (
	_, b, _, _  = runtime.Caller(0)
	ProjectRoot = filepath.Join(filepath.Dir(b))
)

func TestEvents_WorkingDirFile(t *testing.T) {
	fPath := filepath.Join(ProjectRoot, "test_basic_example.txt")

	err := ioutil.WriteFile(fPath, []byte{}, 0700)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(fPath)

	wait := exerciseWatch(t, fPath)

	if err = ioutil.WriteFile(fPath, []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	<-wait
}

func TestEvents_ExistingWorkingDir(t *testing.T) {
	path := filepath.Join(ProjectRoot)

	wait := exerciseWatch(t, path)

	if err := ioutil.WriteFile(filepath.Join(path, "example.txt"), []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(filepath.Join(path, "example.txt"))

	<-wait
}

func TestEvents_SymlinkWorkingDir(t *testing.T) {
	realPath := filepath.Join(ProjectRoot)
	path := filepath.Join(ProjectRoot, "temporary")

	if err := os.Symlink(realPath, path); err != nil {
		t.Fatal(err)
	}

	wait := exerciseWatch(t, path)

	if err := ioutil.WriteFile(filepath.Join(path, "example.txt"), []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(filepath.Join(path, "example.txt"))
		os.Remove(path)
	}()

	<-wait
}

func TestEvents_TempDir(t *testing.T) {
	path, err := ioutil.TempDir("", "fsexample")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(path)

	wait := exerciseWatch(t, path)

	if err = ioutil.WriteFile(filepath.Join(path, "example.txt"), []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	<-wait
}

func exerciseWatch(t *testing.T, path string) <-chan Event {
	t.Logf("exerciseWatch path=%+v", path)

	// path, _ = filepath.EvalSymlinks(path)
	// t.Logf("fullpath:%+v", path)

	dev, err := DeviceForPath(path)
	if err != nil {
		t.Fatal(err)
	}

	es := &EventStream{
		Paths:   []string{path},
		Latency: time.Millisecond,
		Device:  dev,
		Flags:   FileEvents,
	}

	es.Start()

	wait := make(chan Event)

	go func() {
		for msg := range es.Events {
			for _, event := range msg {
				logEvent(t, event)
				wait <- event
				es.Stop()
				return
			}
		}
	}()

	return wait
}

var noteDescription = map[EventFlags]string{
	MustScanSubDirs: "MustScanSubdirs",
	UserDropped:     "UserDropped",
	KernelDropped:   "KernelDropped",
	EventIDsWrapped: "EventIDsWrapped",
	HistoryDone:     "HistoryDone",
	RootChanged:     "RootChanged",
	Mount:           "Mount",
	Unmount:         "Unmount",

	ItemCreated:       "Created",
	ItemRemoved:       "Removed",
	ItemInodeMetaMod:  "InodeMetaMod",
	ItemRenamed:       "Renamed",
	ItemModified:      "Modified",
	ItemFinderInfoMod: "FinderInfoMod",
	ItemChangeOwner:   "ChangeOwner",
	ItemXattrMod:      "XAttrMod",
	ItemIsFile:        "IsFile",
	ItemIsDir:         "IsDir",
	ItemIsSymlink:     "IsSymLink",
}

func logEvent(t *testing.T, event Event) {
	note := ""
	for bit, description := range noteDescription {
		if event.Flags&bit == bit {
			note += description + " "
		}
	}
	t.Logf("EventID: %d Path: %s Flags: %s", event.ID, event.Path, note)
}
