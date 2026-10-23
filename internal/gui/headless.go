//go:build ci

package gui

import (
	"fmt"
	"github.com/qrave1/MusicPort/internal/config"
)

func Run(config.Config) error { return fmt.Errorf("desktop GUI is unavailable in a headless ci build") }
