package gui

import (
	"context"
	"fyne.io/fyne/v2/test"
	"github.com/gojuno/minimock/v3"
	"github.com/qrave1/MusicPort/internal/syncer"
	"image/png"
	"os"
	"testing"
)

func TestTrackFilter(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	service := NewServiceMock(minimock.NewController(t))
	g := New(a, service)
	g.tracks = []syncer.Track{{ID: "one", Title: "First Song"}, {ID: "two", Title: "Second"}}
	g.search.SetText("FIRST")
	if len(g.filtered) != 1 || g.filtered[0].ID != "one" {
		t.Fatal(g.filtered)
	}
	g.search.SetText("two")
	if len(g.filtered) != 1 || g.filtered[0].Title != "Second" {
		t.Fatal(g.filtered)
	}
	g.search.SetText("")
	if len(g.filtered) != 2 {
		t.Fatal(g.filtered)
	}
	if path := os.Getenv("MUSICPORT_PREVIEW"); path != "" {
		g.window.Show()
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, g.window.Canvas().Capture()); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
func TestBusyPreventsSecondRequest(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := New(a, NewServiceMock(minimock.NewController(t)))
	g.busy = true
	g.start("busy", func(context.Context) (func(), error) { t.Error("second request started"); return nil, nil })
	if !g.busy {
		t.Fatal("busy state lost")
	}
}
