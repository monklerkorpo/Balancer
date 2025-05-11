package main

import (
    "github.com/mk/loadBalancer/internal/app"
    _ "modernc.org/sqlite"
)

func main() {
    app.Run()
}
