package main

import "fmt"

// Print base notes path
func printNotesDirPath() error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("Base notes path: \n \t %s \n", cfg.NotesDir)

	return nil
}
