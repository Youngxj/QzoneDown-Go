package main

import (
	"fyne.io/fyne/v2"
)

// OnChangeFuncs is a slice of functions that can be registered
// to run when the user switches tutorial.
var OnChangeFuncs []func()

// Tutorial defines the data structure for a tutorial
type Tutorial struct {
	Title, Intro string
	View         func(w fyne.Window) fyne.CanvasObject
}

type GeneratedTutorial struct {
	title string

	content []string
	code    []func() fyne.CanvasObject
}

var (
	// Tutorials defines the metadata for each tutorial
	Tutorials = map[string]Tutorial{
		"list": {"相册列表",
			"",
			imagesList,
		},
		"config": {"参数配置",
			"",
			configBox,
		},
	}

	// TutorialIndex  defines how our tutorials should be laid out in the index tree
	TutorialIndex = map[string][]string{
		"": {"list", "config"},
	}
)
