//go:build !ci

package gui

import (
	"fyne.io/fyne/v2/app"
	"github.com/qrave1/MusicPort/internal/config"
)

func Run(settings config.Config) error {
	New(app.NewWithID("com.qrave1.musicport"), SpotifyService{Settings: settings}).ShowAndRun()
	return nil
}
