// Test program to verify CLI structure
package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	// Test that the CLI executable exists and can show help
	cmd := exec.Command("./cpc-image", "--help")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error running CLI: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("CLI Help Output:")
	fmt.Print(string(output))

	// Test that convert subcommand exists
	cmd2 := exec.Command("./cpc-image", "convert", "--help")
	output2, err2 := cmd2.Output()
	if err2 != nil {
		fmt.Printf("Error running convert subcommand: %v\n", err2)
		os.Exit(1)
	}

	fmt.Println("\nConvert Subcommand Help:")
	fmt.Print(string(output2))

	fmt.Println("\nCLI structure verification completed successfully!")
}