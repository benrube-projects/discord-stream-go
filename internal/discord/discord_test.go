package discord

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/jonas747/dca"
)

func TestGeneratorDca(t *testing.T) {
	fmt.Println("test")

	// Encoding a file and saving it to disk
	encodeSession, err := dca.EncodeFile("/Users/ben/Downloads/zone.mp3", dca.StdEncodeOptions)
	if err != nil {
		t.Fatal(err)
	}
	// Make sure everything is cleaned up, that for example the encoding process if any issues happened isnt lingering around
	defer encodeSession.Cleanup()

	output, err := os.Create("zone.dca")
	if err != nil {
		fmt.Println("test")

		t.Fatal(err)
	}
	io.Copy(output, encodeSession)
}
