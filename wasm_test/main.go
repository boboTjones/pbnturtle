package main

import (
	"fmt"
	"syscall/js"
)

func main() {
	fmt.Println("Go WebAssembly initialized!")

	// Register a function that JavaScript can call
	js.Global().Set("goHello", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		name := args[0].String()
		return fmt.Sprintf("Hello from Go, %s!", name)
	}))

	// Keep the Go program running
	<-make(chan bool)
}
