package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
)

func main() {
	outFile := flag.String("out", "", "Optional: save private key to file")
	flag.Parse()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate key: %v\n", err)
		os.Exit(1)
	}

	privB64 := base64.StdEncoding.EncodeToString(priv)
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	fmt.Println("=== Ed25519 Key Pair Generated ===")
	fmt.Println()
	fmt.Println("Private Key (use with 'sign' command):")
	fmt.Println(privB64)
	fmt.Println()
	fmt.Println("Public Key (update in pkg/signing/signing.go):")
	fmt.Println(pubB64)

	if *outFile != "" {
		if err := os.WriteFile(*outFile, []byte(privB64), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write key file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nPrivate key saved to: %s\n", *outFile)
	}
}
