package main

import (
	"fmt"
	"github.com/felipe/zemeow/internal/config"
)

func main() {
	fmt.Println("Testing config load...")
	
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Config load failed: %v\n", err)
		return
	}
	
	fmt.Printf("Config loaded successfully: %+v\n", cfg)
}
